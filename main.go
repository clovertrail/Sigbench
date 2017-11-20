package main

import (
	"microsoft.com/sigbench"
	"time"
	"flag"
	"net/rpc"
	"net"
	"log"
	"net/http"
	"strings"
)

func startAsMaster(agents []string) {
	if len(agents) == 0 {
		log.Fatalln("No agents specified")
	}
	log.Println("Agents: ", agents)

	c := &sigbench.MasterController{}

	for _, agent := range agents {
		c.RegisterAgent(agent)
	}

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

	j := &sigbench.Job{
		Phases: []sigbench.JobPhase{
			sigbench.JobPhase{
				Name: "Echo",
				UsersPerSecond: 1000,
				Duration: 10 * time.Second,
			},
		},
		SessionNames: []string{
			"signalrcore:echo",
		},
		SessionPercentages: []float64{
			1,
		},
	}
	c.Run(j)
}

func startAsAgent(address string) {
	controller := &sigbench.AgentController{
	}
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
	var listenAddress = flag.String("l", ":7000", "Listen address")
	var agents = flag.String("agents", "", "Agent addresses separated by comma")

	flag.Parse()

	if isMaster != nil && *isMaster {
		log.Println("Start as master")
		startAsMaster(strings.Split(*agents, ","))
	} else {
		log.Println("Start as agent: ", *listenAddress)
		startAsAgent(*listenAddress)
	}
}

