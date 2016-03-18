package commonlib

import (
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
)

func UnicodeEnCode(code string) string {
	rs := []rune(code)
	json := ""
	html := ""
	for _, r := range rs {
		rint := int(r)
		if rint < 128 {
			json += string(r)
			html += string(r)
		} else {
			json += "\\u" + strconv.FormatInt(int64(rint), 16) // json
			html += "&#" + strconv.Itoa(int(r)) + ";"          // 网页
		}
	}
	return json
}

func UrlEncode(s string) string {
	s1 := url.QueryEscape(s)
	return strings.Replace(s1, "+", "%20", -1)
}

func UrlDecode(s string) string {
	sliceNew := []string{}
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			i++
			//得到每俩个16进制数(一字节)
			s := Substr(s, i, 2)
			j, _ := strconv.ParseInt(s, 16, 0)
			sliceNew = append(sliceNew, string(j))
			i++
		} else {
			sliceNew = append(sliceNew, Substr(s, i, 1))
		}
	}
	return strings.Join(sliceNew, "")
}

//生成32位md5字串
func GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
