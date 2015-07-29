package sender

import (
	"github.com/open-falcon/transfer/g"
	cpool "github.com/open-falcon/transfer/sender/conn_pool"
	nset "github.com/toolkits/container/set"
)

func initConnPools() {
	cfg := g.Config()

	judgeInstances := nset.NewStringSet()
	for _, instance := range cfg.Judge.Cluster {
		judgeInstances.Add(instance)
	}
	JudgeConnPools = cpool.CreateSafeRpcConnPools(cfg.Judge.MaxConns, cfg.Judge.MaxIdle,
		cfg.Judge.ConnTimeout, cfg.Judge.CallTimeout, judgeInstances.ToSlice())

	// graph
	graphInstances := nset.NewSafeSet()
	for _, nitem := range cfg.Graph.Cluster2 {
		for _, addr := range nitem.Addrs {
			graphInstances.Add(addr)
		}
	}
	GraphConnPools = cpool.CreateSafeRpcConnPools(cfg.Graph.MaxConns, cfg.Graph.MaxIdle,
		cfg.Graph.ConnTimeout, cfg.Graph.CallTimeout, graphInstances.ToSlice())

	// graph migrating
	if cfg.Graph.Migrating && cfg.Graph.ClusterMigrating != nil {
		graphMigratingInstances := nset.NewSafeSet()
		for _, cnode := range cfg.Graph.ClusterMigrating2 {
			for _, addr := range cnode.Addrs {
				graphMigratingInstances.Add(addr)
			}
		}
		GraphMigratingConnPools = cpool.CreateSafeRpcConnPools(cfg.Graph.MaxConns, cfg.Graph.MaxIdle,
			cfg.Graph.ConnTimeout, cfg.Graph.CallTimeout, graphMigratingInstances.ToSlice())
	}

	// rrd
	rrdInstances := nset.NewStringSet()
	for _, instance := range cfg.Rrd.Cluster {
		rrdInstances.Add(instance)
	}
	// protocal: binary, json, simplejson, compact
	RrdConnPools = cpool.NewThriftConnPools(cfg.Rrd.MaxConns, cfg.Rrd.MaxIdle,
		cfg.Rrd.ConnTimeout, cfg.Rrd.CallTimeout, rrdInstances.ToSlice(), "binary")

}

func DestroyConnPools() {
	JudgeConnPools.Destroy()
	GraphConnPools.Destroy()
	GraphMigratingConnPools.Destroy()
	RrdConnPools.Destroy()
}
