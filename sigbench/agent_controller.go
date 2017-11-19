package sigbench

import (
)
import (
	"time"
	"log"
	"fmt"
)

type AgentController struct {

}

type AgentRunArgs struct {
	Job Job
	AgentCount int
}

type AgentRunResult struct {
	Error error
}

func (c *AgentController) runPhase(job *Job, phase *JobPhase, agentCount int) {
	for idx, sessionName := range job.SessionNames {
		sessionUsers := int64(float64(phase.UsersPerSecond) * job.SessionPercentages[idx] / float64(agentCount))
		log.Println(fmt.Sprintf("Session %s users: %d", sessionName, sessionUsers))

		var session Session
		if s, ok := SessionMap[sessionName]; ok {
			session = s
		} else {
			log.Fatalln("Session not found: " + sessionName)
		}

		for i := int64(0); i < sessionUsers; i++ {
			go func() {
				ctx := &SessionContext{
					Phase: phase.Name,
				}

				// TODO: Check error
				session.Execute(ctx);
			}()
		}
	}
}

func (c *AgentController) Run(args *AgentRunArgs, result *AgentRunResult) error {
	log.Println(args)
	for _, phase := range args.Job.Phases {
		log.Println("Phase: ", phase)
		start := time.Now()

		ticker := time.NewTicker(time.Second)
		for now := range ticker.C {
			if phase.Duration - now.Sub(start) <= 0 {
				ticker.Stop()
				break
			}

			go c.runPhase(&args.Job, &phase, args.AgentCount)

			if phase.Duration - now.Sub(start) <= 0 {
				ticker.Stop()
				break
			}
		}
	}

	return nil
}

type AgentSetupArgs struct {

}

type AgentSetupResult struct {

}

func (c *AgentController) Setup(args *AgentSetupArgs, result *AgentSetupResult) error {
	for _, session := range SessionMap {
		if err := session.Setup(); err != nil {
			return err
		}
	}

	return nil
}

type AgentListCountersArgs struct {
}

type AgentListCountersResult struct {
	Counters map[string]int64
}

func (c *AgentController) ListCounters(args *AgentListCountersArgs, result *AgentListCountersResult) error {
	result.Counters = make(map[string]int64)
	for _, session := range SessionMap {
		counters := session.Counters()
		for k, v := range counters {
			result.Counters[k] = result.Counters[k] + v
		}
	}
	return nil
}
