package commonlib

import (
	"math"
)

type Message struct {
	Result  string      `required:"true"  description:"success/fail"`
	Message string      `required:"true"  description:"文字信息提示"`
	Data    interface{} `required:"false" description:"如果发生错误，该属性不出现，正常情况下为具体的数据结构"`
	Pager   interface{} `required:"false" description:"如果没有分页数据，该属性不出现"`
}

type Pager struct {
	PageId     int `json:"pageId"`
	RecPerPage int `json:"recPerPage"`
	Total      int `json:"total"`
}

func buildMessage(result bool, message string, data interface{}, pager *Pager) map[string]interface{} {
	msg := make(map[string]interface{})

	if result {
		msg["code"] = 0
	} else {
		msg["code"] = -1
	}

	if data != nil {
		msg["content"] = data
	}
	msg["message"] = message

	if pager != nil {
		msg["pager"] = pager
	}

	return msg
}

func buildPager(pageId, recPerPage, total int) *Pager {

	if pageId < 1 {
		pageId = 1
	}

	if recPerPage < 1 {
		recPerPage = 10
	}

	totalPage := int(math.Ceil(float64(total) / float64(recPerPage)))

	if pageId > totalPage {
		pageId = totalPage
	}

	if total == 0 {
		pageId = 1
	}

	p := new(Pager)

	p.PageId = pageId
	p.RecPerPage = recPerPage
	p.Total = total

	return p
}

func BuildSuccessMessage(message string, data interface{}) map[string]interface{} {
	return buildMessage(true, message, data, nil)
}

func BuildSuccessPageMessage(message string, data interface{}, pager *Pager) map[string]interface{} {
	return buildMessage(true, message, data, pager)
}

func BuildCommonErrorMessage(message string) map[string]interface{} {
	return buildMessage(false, message, nil, nil)
}

func BuildDbErrorMessage(message string) map[string]interface{} {
	return buildMessage(false, message, nil, nil)
}

func BuildParamsErrorMessage() map[string]interface{} {
	return buildMessage(false, "参数格式异常", nil, nil)
}

func BuildObjectNotFountMessage() map[string]interface{} {
	return buildMessage(false, "对象不存在", nil, nil)
}
