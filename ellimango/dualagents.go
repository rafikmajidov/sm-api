package ellimango

type DualAgents struct {
	Oracle Oracle
	Redis  Redis
}

// get all dual agents
func (da *DualAgents) Get() map[string]map[int]string {
	h := Helper{Env: da.Oracle.Env}
	// get dual agents from redis
	redisDualAgents := da.Redis.GetDualAgents()
	// no dual agents from redis, get them from oracle
	// while getting dual agents from oracle, save them in redis
	if len(redisDualAgents) == 0 {
		h.Debug("No redis dual agents, let's get from oracle")
		oracleDualAgents, pIdSids := da.Oracle.GetDualAgents()
		if len(pIdSids) > 0 {
			go da.Redis.WorkerSaveAllDualAgents(pIdSids)
		}
		return oracleDualAgents
	} else {
		return redisDualAgents
	}
}
