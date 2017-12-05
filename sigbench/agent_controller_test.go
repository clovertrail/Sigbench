package sigbench

import "testing"

func TestGetSessionUsers(t *testing.T) {
	c := &AgentController{}

	// 29 = 6 + 6 + 6 + 6 + 5
	if c.getSessionUsers(&JobPhase{UsersPerSecond: 29}, 1, 5, 0) != 6 {
		t.Fatal("29: Major should be 6")
	}
	if c.getSessionUsers(&JobPhase{UsersPerSecond: 29}, 1, 5, 4) != 5 {
		t.Fatal("29: Last should be 5")
	}

	// 30 = 6 + 6 + 6 + 6 + 6
	if c.getSessionUsers(&JobPhase{UsersPerSecond: 30}, 1, 5, 0) != 6 {
		t.Fatal("30: Major should be 6")
	}
	if c.getSessionUsers(&JobPhase{UsersPerSecond: 30}, 1, 5, 4) != 6 {
		t.Fatal("30: Last should be 6")
	}

	// 31 = 7 + 7 + 7 + 7 + 3
	if c.getSessionUsers(&JobPhase{UsersPerSecond: 31}, 1, 5, 0) != 7 {
		t.Fatal("31: Major should be 7")
	}
	if c.getSessionUsers(&JobPhase{UsersPerSecond: 31}, 1, 5, 4) != 3 {
		t.Fatal("31: Last should be 3")
	}

}
