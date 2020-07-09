package ellimango

type Agent struct {
	Oracle Oracle
	Redis  Redis
}

// get agent id by third party id
func (agent *Agent) GetAgentIdByThirdPartyId(deAgentId string) (uint64, error) {
	agentId, err := agent.Oracle.GetAgentIdByThirdPartyId(deAgentId)
	return agentId, err
}

// get agent by id
func (agent *Agent) GetByAgentId(agentId uint64) (FoAgtInfo, error) {
	foAgtInfo, err := agent.Oracle.GetAgentByAgentId(agentId)
	return foAgtInfo, err
}

// get agent by user id
func (agent *Agent) GetByUserId(userId string) (FoAgtInfo, error) {
	foAgtInfo, err := agent.Oracle.GetAgentByUserId(userId)
	return foAgtInfo, err
}
