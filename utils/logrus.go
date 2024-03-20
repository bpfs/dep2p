package utils

import (
	"fmt"
	"runtime"
)

// WhereAmI 返回调用它的函数的文件名和行号
func WhereAmI(depthList ...int) string {
	var depth int
	if depthList == nil {
		depth = 1
	} else {
		depth = depthList[0]
	}
	_, file, line, _ := runtime.Caller(depth)
	return fmt.Sprintf("%s:%d", file, line)
}
