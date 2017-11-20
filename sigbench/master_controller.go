package sigbench

import (
// "errors"
)
import (
	"log"
	"sort"
	"strconv"
	"sync"
	"time"
)

type MasterController struct {
	Agents []*AgentDelegate
}

func (c *MasterController) RegisterAgent(address string) error {
	if agentDelegate, err := NewAgentDelegate(address); err == nil {
		c.Agents = append(c.Agents, agentDelegate)
		return nil
	} else {
		return err
	}
}

func (c *MasterController) setupAllAgents() error {
	var wg sync.WaitGroup

	for _, agent := range c.Agents {
		wg.Add(1)
		go func(agent *AgentDelegate) {
			args := &AgentSetupArgs{}
			var result AgentSetupResult
			if err := agent.Client.Call("AgentController.Setup", args, &result); err != nil {
				// TODO: Report error
				log.Fatalln(err)
			}
			wg.Done()
		}(agent)
	}

	wg.Wait()
	return nil
}

func (c *MasterController) collectCounters() map[string]int64 {
	counters := make(map[string]int64)
	for _, agent := range c.Agents {
		args := &AgentListCountersArgs{}
		var result AgentListCountersResult
		if err := agent.Client.Call("AgentController.ListCounters", args, &result); err != nil {
			log.Println("ERROR: Fail to list counters from agent:", agent.Address, err)
		}
		for k, v := range result.Counters {
			counters[k] = counters[k] + v
		}
	}
	return counters
}

func (c *MasterController) watchCounters(stopChan chan struct{}) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			c.printCounters(c.collectCounters())
		case <-stopChan:
			ticker.Stop()
			return
		}
	}
}

func (c *MasterController) printCounters(counters map[string]int64) {
	table := make([][2]string, 0, len(counters))
	for k, v := range counters {
		table = append(table, [2]string{k, strconv.FormatInt(v, 10)})
	}

	sort.Slice(table, func(i, j int) bool {
		return table[i][0] < table[j][0]
	})

	log.Println("Counters:")
	for _, row := range table {
		log.Println("    ", row[0], ": ", row[1])
	}
}

func (c *MasterController) Run(job *Job) error {
	// TODO: Validate job
	var wg sync.WaitGroup
	var agentCount int = len(c.Agents)

	if err := c.setupAllAgents(); err != nil {
		return err
	}

	for _, agent := range c.Agents {
		wg.Add(1)
		go func(agent *AgentDelegate) {
			args := &AgentRunArgs{
				Job:        *job,
				AgentCount: agentCount,
			}
			var result AgentRunResult
			if err := agent.Client.Call("AgentController.Run", args, &result); err != nil {
				// TODO: report error
				log.Println(err)
			}

			wg.Done()
		}(agent)
	}

	stopWatchCounterChan := make(chan struct{})
	go c.watchCounters(stopWatchCounterChan)

	wg.Wait()

	close(stopWatchCounterChan)

	log.Println("--- Finished ---")
	c.printCounters(c.collectCounters())

	return nil
}
