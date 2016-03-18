package commonlib

import "strconv"

//从指定位置开始截取固定长度
func Substr(str string, start, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}

	return string(rs[start:end])
}

//截取索引号之间的字符内容
func SubstrByStEd(str string, start, end int) string {

	rs := []rune(str)

	return string(rs[start:end])
}

func String2Int(str string) int {
	val, err := strconv.Atoi(str)
	if err != nil {
		val = -1
	}

	return val
}
