package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
)

type Counters struct {
	InProgress  int `json:"signalrcore:echo:inprogress"`
	Established int `json:"signalrcore:echo:established"`
	Error       int `json:"signalrcore:echo:error"`
	Success     int `json:"signalrcore:echo:success"`
	Send        int `json:"signalrcore:echo:msgsendcount"`
	Recv        int `json:"signalrcore:echo:msgrecvcount"`
	SendSize    int `json:"signalrcore:echo:msgsendsize"`
	RecvSize    int `json:"signalrcore:echo:msgrecvsize"`
	LT_200      int `json:"signalrcore:echo:latency:lt_200"`
	LT_400      int `json:"signalrcore:echo:latency:lt_400"`
	LT_600      int `json:"signalrcore:echo:latency:lt_600"`
	LT_800      int `json:"signalrcore:echo:latency:lt_800"`
	LT_1000     int `json:"signalrcore:echo:latency:lt_1000"`
	LT_1200     int `json:"signalrcore:echo:latency:lt_1200"`
	GE_1200     int `json:"signalrcore:echo:latency:ge_1200"`
}

type Monitor struct {
	Timestamp int `json:"Time"`
	Counters  Counters
}

func main() {
	var infile = flag.String("input", "", "Specify the input file")
	var calPercent bool
	calPercent = false
	flag.BoolVar(&calPercent, "perc", false, "Calculate the percentage of caounter distribution")
	flag.Usage = func() {
		fmt.Println("-input <input_file> : specify the input file")
		fmt.Println("-perc               : print percentage of counters")
	}
	flag.Parse()
	if infile == nil || *infile == "" {
		fmt.Println("No input")
		return
	}
	raw, err := ioutil.ReadFile(*infile)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	var monitors []Monitor
	json.Unmarshal(raw, &monitors)
	for _, v := range monitors {
		if calPercent {
			fmt.Printf("timestamp=%d succ=%d err=%d inprogress=%d %d %d %d %d %d %d %d %d %d %d %d\n",
				v.Timestamp, v.Counters.Success, v.Counters.Error, v.Counters.InProgress,
				v.Counters.Send,
				v.Counters.Recv,
				v.Counters.SendSize,
				v.Counters.RecvSize,
				v.Counters.LT_200,
				v.Counters.LT_400,
				v.Counters.LT_600,
				v.Counters.LT_800,
				v.Counters.LT_1000,
				v.Counters.LT_1200,
				v.Counters.GE_1200)
		} else {
			fmt.Printf("timestamp=%d succ=%d err=%d inprogress=%d send=%d recv=%d sendSize=%d recvSize=%d lt_200=%d lt_400=%d lt_600=%d lt_800=%d lt_1000=%d lt_1200=%d ge_1200=%d\n",
				v.Timestamp, v.Counters.Success, v.Counters.Error, v.Counters.InProgress,
				v.Counters.Send,
				v.Counters.Recv,
				v.Counters.SendSize,
				v.Counters.RecvSize,
				v.Counters.LT_200,
				v.Counters.LT_400,
				v.Counters.LT_600,
				v.Counters.LT_800,
				v.Counters.LT_1000,
				v.Counters.LT_1200,
				v.Counters.GE_1200)
		}
	}
	//fmt.Println(monitors)
}
