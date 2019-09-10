package salmon

import (
	"strings"
	"time"
)

const (
	unknown PrxMethod = iota
	roundrobin
	anycast
	amqpRabbit
)

type IoConfig interface {
}


// Config is composition of structs represents Proxy servers configuration.
type Config struct {
	Port            string        `json:"interface"`
	Upstreams       []Upstream    `json:"upstreams"`
	GraceTimoutStop time.Duration `json:"grace_timout_stop"`
}

// Upstream is set of properties for http Handler from configuration.
type Upstream struct {
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	Back        []string  `json:"backends"`
	ProxyMethod PrxMethod `json:"proxy_method"`
}

// ProxyConfigs is array of servers configurations.
type ProxyConfigs struct {
	Configs []Config `json:"configs"`
}


// Custom struct to enumerate ProxyMethods
type PrxMethod int


// String convert pointer to enum to string.
func (p *PrxMethod) String() string {
	return []string{"unknown", "round-robin", "anycast", "amqpRabbit"}[*p]
}

// UnmarshalJSON used by default for one field type PrxMethod when convert from JSON config file to struct.
func (p *PrxMethod) UnmarshalJSON(b []byte) error {
	switch strings.Trim(string(b), "\"") { // trim '\"' string
	case "round-robin":
		*p = roundrobin 	// *p  - change value in memory cell,  the pointer (p *ProxyMethod) указыает на memory cell  and in this cell change value
	case "anycast":
		*p = anycast
	case "amqpRabbit":
		*p = amqpRabbit
	default:
		*p = unknown
	}
	return nil
}




// MarshalJSON used by default for one field type PrxMethod when convert from struct to string.
func (p *PrxMethod) MarshalJSON() (string, error) {
	switch *p {
	case roundrobin:
		return "round-robin", nil
	case anycast:
		return "anycast", nil
	case amqpRabbit:
		return "amqpRabbit", nil
	default:
		return "unknown", nil
	}
}