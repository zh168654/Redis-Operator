package redis

import (
	"github.com/zh168654/Redis-Operator/pkg/api/redis/v1"
)

// Cluster represents a Redis Cluster
type Cluster struct {
	Name           string
	Namespace      string
	Nodes          map[string]*Node
	Status         v1.ClusterStatus
	NodesPlacement v1.NodesPlacementInfo
	ActionsInfo    ClusterActionsInfo
}

// ClusterActionsInfo use to store information about current action on the Cluster
type ClusterActionsInfo struct {
	NbslotsToMigrate int32
}

// NewCluster builds and returns new Cluster instance
func NewCluster(name, namespace string) *Cluster {
	c := &Cluster{
		Name:      name,
		Namespace: namespace,
		Nodes:     make(map[string]*Node),
	}

	return c
}

// AddNode used to add new Node in the cluster
// if node with the same ID is already present in the cluster
// the previous Node is replaced
func (c *Cluster) AddNode(node *Node) {
	if n, ok := c.Nodes[node.ID]; ok {
		n.Clear()
	}

	c.Nodes[node.ID] = node
}

// GetNodeByID returns a Cluster Node by its ID
// if not present in the cluster return an error
func (c *Cluster) GetNodeByID(id string) (*Node, error) {
	if n, ok := c.Nodes[id]; ok {
		return n, nil
	}
	return nil, nodeNotFoundedError
}

// GetNodeByIP returns a Cluster Node by its ID
// if not present in the cluster return an error
func (c *Cluster) GetNodeByIP(ip string) (*Node, error) {
	findFunc := func(node *Node) bool {
		return node.IP == ip
	}

	return c.GetNodeByFunc(findFunc)
}

// GetNodeByPodName returns a Cluster Node by its Pod name
// if not present in the cluster return an error
func (c *Cluster) GetNodeByPodName(name string) (*Node, error) {
	findFunc := func(node *Node) bool {
		if node.Pod == nil {
			return false
		}
		if node.Pod.Name == name {
			return true
		}
		return false
	}

	return c.GetNodeByFunc(findFunc)
}

// GetNodeByFunc returns first node found by the FindNodeFunc
func (c *Cluster) GetNodeByFunc(f FindNodeFunc) (*Node, error) {
	for _, n := range c.Nodes {
		if f(n) {
			return n, nil
		}
	}
	return nil, nodeNotFoundedError
}

// GetNodesByFunc returns first node found by the FindNodeFunc
func (c *Cluster) GetNodesByFunc(f FindNodeFunc) (Nodes, error) {
	nodes := Nodes{}
	for _, n := range c.Nodes {
		if f(n) {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		return nodes, nodeNotFoundedError
	}
	return nodes, nil
}

// FindNodeFunc function for finding a Node
// it is use as input for GetNodeByFunc and GetNodesByFunc
type FindNodeFunc func(node *Node) bool

// ToAPIClusterStatus convert the Cluster information to a api
func (c *Cluster) ToAPIClusterStatus() v1.RedisClusterClusterStatus {
	status := v1.RedisClusterClusterStatus{}
	status.Status = c.Status
	for _, node := range c.Nodes {
		status.Nodes = append(status.Nodes, node.ToAPINode())
	}
	return status
}
