package parser

import "strconv"

func ToInt64(in string) (int64, error) {
	return strconv.ParseInt(in, 0, 0)
}

func DlvBreak() {}
