package sigbench

//import "time"

type JobPhase struct {
	Name           string
	UsersPerSecond int64
	Duration       int64
}

type Job struct {
	Phases             []JobPhase
	SessionNames       []string
	SessionPercentages []float64
	SessionParams      map[string]string
}
