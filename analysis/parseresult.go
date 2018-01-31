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
	LT_100      int `json:"signalrcore:echo:latency:lt_100"`
	LT_200      int `json:"signalrcore:echo:latency:lt_200"`
	LT_300      int `json:"signalrcore:echo:latency:lt_300"`
	LT_400      int `json:"signalrcore:echo:latency:lt_400"`
	LT_500      int `json:"signalrcore:echo:latency:lt_500"`
	LT_600      int `json:"signalrcore:echo:latency:lt_600"`
	LT_700      int `json:"signalrcore:echo:latency:lt_700"`
	LT_800      int `json:"signalrcore:echo:latency:lt_800"`
	LT_900      int `json:"signalrcore:echo:latency:lt_900"`
	LT_1000     int `json:"signalrcore:echo:latency:lt_1000"`
	GE_1000     int `json:"signalrcore:echo:latency:ge_1000"`
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
			fmt.Printf("timestamp=%d succ=%d err=%d inprogress=%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d\n",
				v.Timestamp, v.Counters.Success, v.Counters.Error, v.Counters.InProgress,
				v.Counters.Send,
				v.Counters.Recv,
				v.Counters.SendSize,
				v.Counters.RecvSize,
				v.Counters.LT_100,
				v.Counters.LT_200,
				v.Counters.LT_300,
				v.Counters.LT_400,
				v.Counters.LT_500,
				v.Counters.LT_600,
				v.Counters.LT_700,
				v.Counters.LT_800,
				v.Counters.LT_900,
				v.Counters.LT_1000,
				v.Counters.GE_1000)
		} else {
			fmt.Printf("timestamp=%d succ=%d err=%d inprogress=%d send=%d recv=%d sendSize=%d recvSize=%d lt_100=%d lt_200=%d lt_300=%d lt_400=%d lt_500=%d lt_600=%d lt_700=%d lt_800=%d lt_900=%d lt_1000=%d ge_1000=%d\n",
				v.Timestamp, v.Counters.Success, v.Counters.Error, v.Counters.InProgress,
				v.Counters.Send,
				v.Counters.Recv,
				v.Counters.SendSize,
				v.Counters.RecvSize,
				v.Counters.LT_100,
				v.Counters.LT_200,
				v.Counters.LT_300,
				v.Counters.LT_400,
				v.Counters.LT_500,
				v.Counters.LT_600,
				v.Counters.LT_700,
				v.Counters.LT_800,
				v.Counters.LT_900,
				v.Counters.LT_1000,
				v.Counters.GE_1000)
		}
	}
	//fmt.Println(monitors)
}
