package g

import (
	"encoding/json"
	"github.com/toolkits/file"
	"log"
	"strings"
	"sync"
)

type HttpConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

type RpcConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

type SocketConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
	Timeout int    `json:"timeout"`
}

type JudgeConfig struct {
	Enabled     bool                    `json:"enabled"`
	Batch       int                     `json:"batch"`
	ConnTimeout int                     `json:"connTimeout"`
	CallTimeout int                     `json:"callTimeout"`
	PingMethod  string                  `json:"pingMethod"`
	MaxConns    int                     `json:"maxConns"`
	MaxIdle     int                     `json:"maxIdle"`
	Replicas    int                     `json:"replicas"`
	Cluster     map[string]string       `json:"cluster"`
	Cluster2    map[string]*ClusterNode `json:"cluster2"`
}

type GraphConfig struct {
	Enabled           bool                    `json:"enabled"`
	Batch             int                     `json:"batch"`
	ConnTimeout       int                     `json:"connTimeout"`
	CallTimeout       int                     `json:"callTimeout"`
	PingMethod        string                  `json:"pingMethod"`
	MaxConns          int                     `json:"maxConns"`
	MaxIdle           int                     `json:"maxIdle"`
	Replicas          int                     `json:"replicas"`
	Migrating         bool                    `json:"migrating"`
	Cluster           map[string]string       `json:"cluster"`
	ClusterMigrating  map[string]string       `json:"clusterMigrating"`
	Cluster2          map[string]*ClusterNode `json:"cluster2"`
	ClusterMigrating2 map[string]*ClusterNode `json:"clusterMigrating2"`
}

type RrdConfig struct {
	Enabled     bool              `json:"enabled"`
	Batch       int               `json:"batch"`
	ConnTimeout int               `json:"connTimeout"`
	CallTimeout int               `json:"callTimeout"`
	PingMethod  string            `json:"pingMethod"`
	MaxConns    int               `json:"maxConns"`
	MaxIdle     int               `json:"maxIdle"`
	Replicas    int               `json:"replicas"`
	Cluster     map[string]string `json:"cluster"`
	Nodes       []string          `json:"nodes"`
	NodeSize    uint64            `json:"nodeSize"`
}

type GlobalConfig struct {
	Debug  bool          `json:"debug"`
	Http   *HttpConfig   `json:"http"`
	Rpc    *RpcConfig    `json:"rpc"`
	Socket *SocketConfig `json:"socket"`
	Judge  *JudgeConfig  `json:"judge"`
	Graph  *GraphConfig  `json:"graph"`
	Rrd    *RrdConfig    `json:"rrd"`
}

var (
	ConfigFile string
	config     *GlobalConfig
	configLock = new(sync.RWMutex)
)

func Config() *GlobalConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

func ParseConfig(cfg string) {
	if cfg == "" {
		log.Fatalln("use -c to specify configuration file")
	}

	if !file.IsExist(cfg) {
		log.Fatalln("config file:", cfg, "is not existent. maybe you need `mv cfg.example.json cfg.json`")
	}

	ConfigFile = cfg

	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
	}

	// split cluster config
	c.Judge.Cluster2 = formatClusterItems(c.Judge.Cluster)
	c.Graph.Cluster2 = formatClusterItems(c.Graph.Cluster)
	c.Graph.ClusterMigrating2 = formatClusterItems(c.Graph.ClusterMigrating)

	// rrd addrs
	c.Rrd.Nodes = keySliceOfMap(c.Rrd.Cluster)
	c.Rrd.NodeSize = uint64(len(c.Rrd.Nodes))

	// check
	if !checkConfig(c) {
		log.Fatalln("bad cfg")
	}

	configLock.Lock()
	defer configLock.Unlock()
	config = &c

	log.Println("g.ParseConfig ok, file ", cfg)
}

// check
func checkConfig(c GlobalConfig) bool {
	if c.Rrd.Enabled && c.Rrd.NodeSize < 1 {
		return false
	}

	return true
}

// CLUSTER NODE
type ClusterNode struct {
	Addrs []string `json:"addrs"`
}

func NewClusterNode(addrs []string) *ClusterNode {
	return &ClusterNode{addrs}
}

// map["node"]="host1,host2" --> map["node"]=["host1", "host2"]
func formatClusterItems(cluster map[string]string) map[string]*ClusterNode {
	ret := make(map[string]*ClusterNode)
	for node, clusterStr := range cluster {
		items := strings.Split(clusterStr, ",")
		nitems := make([]string, 0)
		for _, item := range items {
			nitems = append(nitems, strings.TrimSpace(item))
		}
		ret[node] = NewClusterNode(nitems)
	}

	return ret
}

func keySliceOfMap(mapv map[string]string) []string {
	ret := make([]string, 0)
	for key, _ := range mapv {
		ret = append(ret, key)
	}
	return ret
}
