package tools

func GetPoint[T any](in T) *T {
	return &in
}
