package salmon
// Struct for Round-Robin counter
type Count struct {
	scorer int
}

// Func to reset count number
func (c *Count) ResetCount() {
	c.scorer = 0
}

// Func to increase count number
func (c *Count) IncreaseCount(){
	c.scorer++
}