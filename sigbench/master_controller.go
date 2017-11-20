package sigbench

import (
	// "errors"
)
import (
	"sync"
	"log"
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

func (c *MasterController) watchCounters(stopChan chan bool) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <- ticker.C:
			log.Println("Counters: ", c.collectCounters())
		case <- stopChan:
			ticker.Stop()
			return
		}
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
				Job: *job,
				AgentCount: agentCount,
			}
			var result AgentRunResult
			if err := agent.Client.Call("AgentController.Run", args, &result); err != nil {
				// TODO: report error
				log.Println(err)
			}

			wg.Done()
		}(agent);
	}

	stopWatchCounterChan := make(chan bool)
	go c.watchCounters(stopWatchCounterChan)

	wg.Wait()

	stopWatchCounterChan <- true

	log.Println("Final counters: ", c.collectCounters())

	return nil;
}