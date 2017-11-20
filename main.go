package main

import (
	"encoding/json"
	"flag"
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
)

func startAsMaster(agents []string, config string, outDir string) {
	if len(agents) == 0 {
		log.Fatalln("No agents specified")
	}
	log.Println("Agents: ", agents)

	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalln(err)
	}
	log.Println("Ouptut directory: ", outDir)

	c := &sigbench.MasterController{
		SnapshotWriter: snapshot.NewJsonSnapshotWriter(outDir + "/counters.txt"),
	}

	for _, agent := range agents {
		c.RegisterAgent(agent)
	}

	var job sigbench.Job
	if f, err := os.Open(config); err == nil {
		decoder := json.NewDecoder(f)
		if err := decoder.Decode(&job); err != nil {
			log.Fatalln("Fail to load config file: ", err)
		}
	} else {
		log.Fatalln("Fail to open config file: ", err)
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

func main() {
	var isMaster = flag.Bool("master", false, "True if master")
	var config = flag.String("config", "config.json", "Job config file")
	var outDir = flag.String("outDir", "output/"+strconv.FormatInt(time.Now().Unix(), 10), "Output directory")
	var listenAddress = flag.String("l", ":7000", "Listen address")
	var agents = flag.String("agents", "", "Agent addresses separated by comma")

	flag.Parse()

	if isMaster != nil && *isMaster {
		log.Println("Start as master")
		startAsMaster(strings.Split(*agents, ","), *config, *outDir)
	} else {
		log.Println("Start as agent: ", *listenAddress)
		startAsAgent(*listenAddress)
	}
}
