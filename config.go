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

// Interface for Server Config
type IoConfig interface {
}

// Config - Composition struct for Server configuration
type Config struct {
	Port            string        `json:"interface"`
	Upstreams       []Upstream    `json:"upstreams"`
	GraceTimoutStop time.Duration `json:"grace_timout_stop"`
}

// Struct for Server Upstream
type Upstream struct {
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	Back        []string  `json:"backends"`
	ProxyMethod PrxMethod `json:"proxy_method"`
}

// Struct include slice of servers configurations
type ProxyConfigs struct {
	Configs []Config `json:"configs"`
}


type PrxMethod int
func (p *PrxMethod) String() string {
	return []string{"unknown", "round-robin", "anycast", "amqpRabbit"}[*p]
}

// UnmarshalJSON(b []byte) used by default for one field type PrxMethod when convert from JSON config file to struct Upstream
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

// MarshalJSON used by default for one field type PrxMethod when convert from struct to Json doc
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