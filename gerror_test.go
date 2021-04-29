package gerror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type CustomFormatter struct{}

func (*CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message), nil
}

var errorMessages = map[int]string{
	400: "Bad Request",
	401: "Unauthorized",
	403: "Forbidden",
	404: "Not Found",
	500: "Internal Server Error",
	503: "Service Unavailable",
}

var buf bytes.Buffer

func init() {
	logrus.SetFormatter(&CustomFormatter{})
	logrus.SetOutput(&buf)
}

func readLog(t *testing.T) string {
	_bytes, err := ioutil.ReadAll(&buf)
	assert.NoError(t, err)
	return string(_bytes)
}

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseBody(t *testing.T, res *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	body, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)
	return response
}

var pathI int

func getTestPath() string {
	pathI++
	return fmt.Sprintf("/route-%d", pathI)
}

func TestCustomResponseFunction(t *testing.T) {
	router := gin.New()

	router.Use(Middleware(MiddlewareOption{
		ResponseBodyFunc: func(code int, message string) interface{} {
			return gin.H{
				"foo":           "foo",
				"bar":           "bar",
				"code":          code,
				"customMessage": message,
			}
		},
	}))
	path := getTestPath()
	router.GET(path, func(c *gin.Context) {
		AbortWithHint(c, 500, "custom hint")
	})
	res := performRequest(router, "GET", path)
	assert.Equal(t, res.Code, 500)
	body := parseBody(t, res)
	assert.Equal(t, "custom hint", body["customMessage"])
	assert.Equal(t, float64(500), body["code"])
}

// Log the error whose status code >= 400
func TestCustomErrorFunction(t *testing.T) {
	router := gin.New()
	router.Use(Middleware(MiddlewareOption{
		LoggingFunc: func(code int, err error) {
			if code >= 400 {
				logrus.Errorln(err)
			}
		},
	}))
	path := getTestPath()
	router.GET(path, func(c *gin.Context) {
		err := errors.New("error")
		AbortWithError(c, 400, err)
	})
	res := performRequest(router, "GET", path)
	assert.Equal(t, res.Code, 400)
	assert.Equal(t, "error", readLog(t))
}

func TestDefaultOptionFunction(t *testing.T) {
	router := gin.New()
	router.Use(Middleware(MiddlewareOption{}))

	t.Run("single error", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			err := errors.New("single error")
			AbortWithErrorAndHint(c, 500, err, "custom hint")
		})
		res := performRequest(router, "GET", path)
		assert.Equal(t, res.Code, 500)
		assert.Equal(t, "single error", readLog(t))
		body := parseBody(t, res)
		assert.Equal(t, "custom hint", body["message"])
	})
	t.Run("multiple statusCode and errors", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			err := errors.New("custom error")
			AbortWithErrorAndHint(c, 500, err, "error1")
			AbortWithErrorAndHint(c, 400, err, "error2")
		})
		res := performRequest(router, "GET", path)
		// It uses the last error
		assert.Equal(t, res.Code, 400)
		body := parseBody(t, res)
		assert.Equal(t, "error2", body["message"])
	})
	t.Run("without err", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			AbortWithError(c, 500, nil)
		})
		res := performRequest(router, "GET", path)
		assert.Equal(t, res.Code, 500)
	})
	t.Run("abort with origin error", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			_ = c.AbortWithError(500, errors.New("origin error"))
		})
		res := performRequest(router, "GET", path)
		assert.Equal(t, res.Code, 500)
		body := parseBody(t, res)
		assert.Equal(t, "origin error", body["message"])
	})
}

func TestCustomResponseBodyFunc(t *testing.T) {
	router := gin.New()
	router.Use(Middleware(MiddlewareOption{ResponseBodyFunc: func(code int, message string) interface{} {
		if message == "" {
			if errorMessages[code] != "" {
				return gin.H{"message": errorMessages[code]}
			} else {
				return nil
			}
		} else {
			return gin.H{"message": message}
		}
	}}))
	t.Run("defined status code", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			AbortWithHint(c, 403, "")
		})
		res := performRequest(router, "GET", path)
		assert.Equal(t, res.Code, 403)
		body := parseBody(t, res)
		assert.Equal(t, "Forbidden", body["message"])
	})
	t.Run("only status code", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			AbortWithHint(c, 402, "")
		})
		res := performRequest(router, "GET", path)
		assert.Equal(t, res.Code, 402)
		_bytes, err := ioutil.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.Len(t, _bytes, 0)
	})
}
