package utils

// FibonacciArray 创建一个长度为 n 的斐波那契数组。
func FibonacciArray(n int) []int64 {
	res := make([]int64, n)
	for i := 0; i < n; i++ {
		if i <= 1 {
			res[i] = 1
		} else {
			res[i] = res[i-1] + res[i-2]
		}
	}
	return res
}
