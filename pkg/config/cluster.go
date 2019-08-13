package config

import "github.com/spf13/pflag"

// Cluster used to store all Redis Cluster configuration information
type Cluster struct {
	Name                string
	Namespace           string
	NodeService         string
	NodeServiceType     string
	NodeServiceNodePort string
}

// AddFlags use to add the Redis-Cluster Config flags to the command line
func (c *Cluster) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Name, "name", "", "redis-cluster name")
	fs.StringVar(&c.Namespace, "ns", "", "redis-node k8s namespace")
	fs.StringVar(&c.NodeService, "rs", "", "redis-node k8s service name")
	fs.StringVar(&c.NodeServiceType, "rstype", "", "redis-node k8s service type")
	fs.StringVar(&c.NodeServiceNodePort, "rsnodeport", "", "redis-node k8s service nodePort start")

}
