package commonlib

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func HttpGet(url string) ([]byte, error) {

	Log.Trace("Http Get:" + url)

	client := &http.Client{
		CheckRedirect: nil,
	}

	reqest, _ := http.NewRequest("GET", url, nil)

	reqest.Header.Set("User-Agent", " Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36")
	reqest.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	reqest.Header.Set("Accept-Charset", "GBK,utf-8;q=0.7,*;q=0.3")
	reqest.Header.Set("Accept-Encoding", "gzip,deflate,sdch")
	reqest.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	reqest.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	reqest.Header.Set("Cache-Control", "max-age=0")
	reqest.Header.Set("Connection", "keep-alive")
	reqest.Header.Set("Referer", url)

	resp, err := client.Do(reqest)

	if err != nil {
		Log.Error("Http Get:", url, "发生错误:", err)
		return nil, err
	}

	defer resp.Body.Close()

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			Log.Error("Http Get:", url, "发生错误:", err)
			return nil, err
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	if reader != nil {
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			Log.Error("Http Get:", url, "发生错误:", err)
			return nil, err
		}
		return body, nil
	}

	return nil, nil
}

func HttpPost(url, postStr string) ([]byte, error) {

	Log.Trace("Http POST :" + url + ",body:" + postStr)

	client := &http.Client{
		CheckRedirect: nil,
	}

	postBytesReader := bytes.NewReader([]byte(postStr))
	reqest, _ := http.NewRequest("POST", url, postBytesReader)

	reqest.Header.Set("User-Agent", " Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36")
	reqest.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	reqest.Header.Set("Accept-Charset", "GBK,utf-8;q=0.7,*;q=0.3")
	reqest.Header.Set("Accept-Encoding", "gzip,deflate,sdch")
	reqest.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	reqest.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	reqest.Header.Set("Cache-Control", "max-age=0")
	reqest.Header.Set("Connection", "keep-alive")
	reqest.Header.Set("Referer", url)

	resp, err := client.Do(reqest)

	if err != nil {
		Log.Error("Http POST :", url, "发生错误:", err)
		return nil, err
	}

	defer resp.Body.Close()

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			Log.Error("Http POST :", url, "发生错误:", err)
			return nil, err
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	if reader != nil {
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			Log.Error("Http POST :", url, "发生错误:", err)
			return nil, err
		}
		return body, nil
	}

	return nil, nil
}

func HttpPostFile(url string, params map[string]string, paramName, path string) ([]byte, error) {

	client := &http.Client{
		CheckRedirect: nil,
	}

	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))

	if err != nil {
		Log.Error("Http POST File :", url, "发生错误:", err)
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err = writer.Close()

	if err != nil {
		Log.Error("Http POST File :", url, "发生错误:", err)
		return nil, err
	}

	request, _ := http.NewRequest("POST", url, body)

	request.Header.Set("User-Agent", " Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("Accept-Charset", "GBK,utf-8;q=0.7,*;q=0.3")
	request.Header.Set("Accept-Encoding", "gzip,deflate,sdch")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	request.Header.Set("Cache-Control", "max-age=0")
	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("Referer", url)

	resp, err := client.Do(request)

	if err != nil {
		Log.Error("Http POST File :", url, "发生错误:", err)
		return nil, err
	}

	defer resp.Body.Close()

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			Log.Error("Http POST File :", url, "发生错误:", err)
			return nil, err
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	if reader != nil {
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			Log.Error("Http POST File :", url, "发生错误:", err)
			return nil, err
		}
		return body, nil
	}

	return nil, nil
}
