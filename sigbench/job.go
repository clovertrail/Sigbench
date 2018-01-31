package sigbench

//import "time"
import (
//"gopkg.in/yaml.v2"
)

type JobPhase struct {
	Name           string `yaml:"Name"`
	UsersPerSecond int64  `yaml:"UsersPerSecond"`
	Duration       int64  `yaml:"Duration"`
}

type Job struct {
	Phases             []JobPhase        `yaml:"Phases,flow"`
	SessionNames       []string          `yaml:"SessionNames,flow"`
	SessionPercentages []float64         `yaml:"SessionPercentages,flow"`
	SessionParams      map[string]string `yaml:"SessionParams"`
}
