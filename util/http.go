package util

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"github.com/sath33sh/infra/log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func HttpHead(url string) (resp *http.Response, err error) {
	c := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err = c.Head(url)
	if err != nil {
		log.Errorf("Failed to head %s: %v", url, err)
		return nil, ErrNetAccess
	}

	return resp, nil
}

func HttpGet(url string) (resp *http.Response, err error) {
	c := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err = c.Get(url)
	if err != nil {
		log.Errorf("Failed to get %s: %v", url, err)
		return nil, ErrNetAccess
	}

	return resp, nil
}

func HttpJsonGet(url string, result interface{}) (err error) {
	c := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	var resp *http.Response
	resp, err = c.Get(url)
	if err != nil {
		log.Errorf("Failed to get %s: %v", url, err)
		return ErrNetAccess
	}

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
		log.Errorf("Failed to decode %s: %v", url, err)
		return ErrJsonDecode
	}

	return nil
}

func HttpXmlGet(url string, result interface{}) (err error) {
	c := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	var resp *http.Response
	resp, err = c.Get(url)
	if err != nil {
		log.Errorf("Failed to get %s: %v", url, err)
		return ErrNetAccess
	}

	defer resp.Body.Close()

	if err = xml.NewDecoder(resp.Body).Decode(result); err != nil {
		log.Errorf("Failed to decode %s: %v", url, err)
		return ErrXmlDecode
	}

	return nil
}

func HttpGetImage(url string) (data []byte, mediaSubType string, err error) {
	c := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	var resp *http.Response
	resp, err = c.Get(url)
	if err != nil {
		log.Errorf("Failed to get %s: %v", url, err)
		return data, mediaSubType, ErrNetAccess
	}

	defer resp.Body.Close()

	ctype := resp.Header.Get("Content-Type")
	if len(ctype) == 0 {
		// Default to jpeg.
		mediaSubType = "jpeg"
	} else {
		mediaTypeArr := strings.Split(ctype, "/")
		mediaSubType = mediaTypeArr[1]
	}

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read %s: %v", url, err)
		return data, mediaSubType, ErrFileAccess
	}

	return data, mediaSubType, nil
}

func HttpDownload(url, filepath string) (err error) {
	c := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	file, err := os.Create(filepath)
	if err != nil {
		log.Errorf("Failed to create file %s: %v", filepath, err)
		return ErrFileAccess
	}
	defer file.Close()

	resp, err := c.Get(url)
	if err != nil {
		log.Errorf("Failed to get %s: %v", url, err)
		return ErrNetAccess
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Errorf("Failed to copy file %s: %v", filepath, err)
		return ErrFileAccess
	}

	return nil
}

func HttpJsonPost(url string, reqData interface{}, respData interface{}) (err error) {
	c := http.Client{}

	var reqReader *bytes.Reader = nil
	if reqData != nil {
		data, err := json.Marshal(reqData)
		if err != nil {
			log.Errorf("JSON marshal error %s: %v", url, err)
			return ErrInvalidInput
		}

		reqReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", url, reqReader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Errorf("POST failed: URL %s, status %s: %v", url, resp.Status, err)
		return ErrNetAccess
	}

	defer resp.Body.Close()

	if respData != nil {
		if err = json.NewDecoder(resp.Body).Decode(respData); err != nil {
			log.Errorf("Failed to decode %s: %v", url, err)
			return ErrJsonDecode
		}
	}

	return nil
}
