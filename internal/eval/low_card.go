package eval

func compareLowCard(i1, i2 int) int {
	if i1 < i2 {
		return 1
	}
	if i2 < i1 {
		return -1
	}
	return 0
}
