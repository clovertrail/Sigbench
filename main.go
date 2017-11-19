package main

import (
	"microsoft.com/sigbench"
	"time"
	"flag"
	"net/rpc"
	"net"
	"log"
	"net/http"
)

func startAsMaster() {
	c := &sigbench.MasterController{}
	c.RegisterAgent("localhost:7000")

	j := &sigbench.Job{
		Phases: []sigbench.JobPhase{
			sigbench.JobPhase{
				Name: "First",
				UsersPerSecond: 1,
				Duration: 3 * time.Second,
			},
			sigbench.JobPhase{
				Name: "Second",
				UsersPerSecond: 3,
				Duration: 3 * time.Second,
			},
		},
		SessionNames: []string{
			"dummy",
		},
		SessionPercentages: []float64{
			1,
		},
	}
	c.Run(j)
}

func startAsAgent() {
	controller := &sigbench.AgentController{
	}
	rpc.Register(controller)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":7000")
	if err != nil {
		log.Fatal("Fail to listen:", err)
	}
	http.Serve(l, nil)
}

func main() {
	var isMaster = flag.Bool("master", false, "True if master")
	flag.Parse()
	if (isMaster != nil && *isMaster) {
		log.Println("Start as master")
		startAsMaster()
	} else {
		log.Println("Start as agent")
		startAsAgent()
	}
}

