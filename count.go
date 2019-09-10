package salmon
// Count represents counter for RoundRobin requests between upstreams.
type Count struct {
	scorer int
}

// ResetCount clear current counter.
func (c *Count) ResetCount() {
	c.scorer = 0
}

// IncreaseCount append int 1 to counter.
func (c *Count) IncreaseCount(){
	c.scorer++
}