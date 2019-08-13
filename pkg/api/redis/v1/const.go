package v1

const (
	// ClusterNameLabelKey Label key for the ClusterName
	ClusterNameLabelKey string = "redis-operator.k8s.io/cluster-name"
	// PodNoLabelKey Label key for the ClusterName
	PodNoLabelKey string = "redis-operator.k8s.io/pod-no"
	// PodSpecMD5LabelKey label key for the PodSpec MD5 hash
	PodSpecMD5LabelKey string = "redis-operator.k8s.io/podspec-md5"
)
