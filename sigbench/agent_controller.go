package sigbench

import (
)
import (
	"time"
	"log"
	"fmt"
	"microsoft.com/sigbench/sessions"
	"sync"
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

func (c *AgentController) runPhase(job *Job, phase *JobPhase, agentCount int, wg *sync.WaitGroup) {
	for idx, sessionName := range job.SessionNames {
		sessionUsers := int64(float64(phase.UsersPerSecond) * job.SessionPercentages[idx] / float64(agentCount))
		log.Println(fmt.Sprintf("Session %s users: %d", sessionName, sessionUsers))

		var session sessions.Session
		if s, ok := sessions.SessionMap[sessionName]; ok {
			session = s
		} else {
			log.Fatalln("Session not found: " + sessionName)
		}

		for i := int64(0); i < sessionUsers; i++ {
			wg.Add(1)
			go func() {
				ctx := &sessions.SessionContext{
					Phase: phase.Name,
				}

				// TODO: Check error
				session.Execute(ctx);

				// Done for user
				wg.Done()
			}()
		}
	}

	// Done for phase
	wg.Done()
}

func (c *AgentController) Run(args *AgentRunArgs, result *AgentRunResult) error {
	log.Println(args)
	var wg sync.WaitGroup

	for _, phase := range args.Job.Phases {
		log.Println("Phase: ", phase)
		start := time.Now()

		ticker := time.NewTicker(time.Second)
		for now := range ticker.C {
			if phase.Duration - now.Sub(start) <= 0 {
				ticker.Stop()
				break
			}

			wg.Add(1)
			go c.runPhase(&args.Job, &phase, args.AgentCount, &wg)

			if phase.Duration - now.Sub(start) <= 0 {
				ticker.Stop()
				break
			}
		}
	}

	wg.Wait()

	return nil
}

type AgentSetupArgs struct {

}

type AgentSetupResult struct {

}

func (c *AgentController) Setup(args *AgentSetupArgs, result *AgentSetupResult) error {
	for _, session := range sessions.SessionMap {
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
	for _, session := range sessions.SessionMap {
		counters := session.Counters()
		for k, v := range counters {
			result.Counters[k] = result.Counters[k] + v
		}
	}
	return nil
}
