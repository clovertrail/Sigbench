package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	"microsoft.com/sigbench"
	"microsoft.com/sigbench/snapshot"
	"microsoft.com/sigbench/service"
	"gopkg.in/yaml.v2"
)

func startAsMaster(agents []string, config string, outDir string) {
	if len(agents) == 0 {
		log.Fatalln("No agents specified")
	}
	log.Println("Agents: ", agents)

	// Create output directory
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalln(err)
	}
	log.Println("Ouptut directory: ", outDir)

	c := &sigbench.MasterController{
		SnapshotWriter: snapshot.NewJsonSnapshotWriter(outDir + "/counters.txt"),
	}

	for _, agent := range agents {
		if err := c.RegisterAgent(agent); err != nil {
			log.Fatalln("Fail to register agent: ", agent, err)
			return
		}
	}

	var job sigbench.Job
	if yamlFile, err := ioutil.ReadFile(config); err == nil {
		if err := yaml.Unmarshal(yamlFile, &job); err != nil {
			log.Fatalln("Fail to load config file: ", err)
			return
		}
		copied, err := yaml.Marshal(&job)
		if err != nil {
			log.Fatalln("Fail to encode a copy of config file: ", err)
			return
		}
		if err := ioutil.WriteFile(outDir+"/config.yaml", copied, 0644); err != nil {
			log.Fatalln("Fail to save a copy of config file: ", err)
			return
		}
	} else {
		log.Fatalln("Fail to open config file: ", err)
		return
	}

	c.Run(&job)

	// j := &sigbench.Job{
	// 	Phases: []sigbench.JobPhase{
	// 		sigbench.JobPhase{
	// 			Name: "First",
	// 			UsersPerSecond: 1,
	// 			Duration: 3 * time.Second,
	// 		},
	// 		sigbench.JobPhase{
	// 			Name: "Second",
	// 			UsersPerSecond: 3,
	// 			Duration: 3 * time.Second,
	// 		},
	// 	},
	// 	SessionNames: []string{
	// 		"dummy",
	// 	},
	// 	SessionPercentages: []float64{
	// 		1,
	// 	},
	// }

	// HTTP
	// j := &sigbench.Job{
	// 	Phases: []sigbench.JobPhase{
	// 		sigbench.JobPhase{
	// 			Name: "Get",
	// 			UsersPerSecond: 10,
	// 			Duration: 10 * time.Second,
	// 		},
	// 	},
	// 	SessionNames: []string{
	// 		"http-get",
	// 	},
	// 	SessionPercentages: []float64{
	// 		1,
	// 	},
	// }

	// j := &sigbench.Job{
	// 	Phases: []sigbench.JobPhase{
	// 		sigbench.JobPhase{
	// 			Name: "Echo",
	// 			UsersPerSecond: 1000,
	// 			Duration: 30 * time.Second,
	// 		},
	// 	},
	// 	SessionNames: []string{
	// 		"signalrcore:echo",
	// 	},
	// 	SessionPercentages: []float64{
	// 		1,
	// 	},
	// }
}

func startAsAgent(address string) {
	controller := &sigbench.AgentController{}
	rpc.Register(controller)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Fail to listen:", err)
	}
	http.Serve(l, nil)
}

func startAsService(address string, outDir string) {
	mux := service.NewServiceMux(outDir)
	log.Fatal(http.ListenAndServe(address, mux))
}

func main() {
	var mode = flag.String("mode", "agent", "service | cli | agent")
	var config = flag.String("config", "config.json", "Job config file")
	var outDir = flag.String("outDir", "output/"+strconv.FormatInt(time.Now().Unix(), 10), "Output directory")
	var listenAddress = flag.String("l", ":7000", "Listen address")
	var agents = flag.String("agents", "", "Agent addresses separated by comma")

	flag.Parse()

	if mode == nil {
		log.Fatalln("No mode specified")
	}

	if *mode == "cli" {
		log.Println("Start as CLI master")
		startAsMaster(strings.Split(*agents, ","), *config, *outDir)
	} else if *mode == "service" {
		log.Println("Start as service")
		startAsService(*listenAddress, *outDir)
	} else {
		log.Println("Start as agent: ", *listenAddress)
		startAsAgent(*listenAddress)
	}
}
