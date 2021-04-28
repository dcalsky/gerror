package gerror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
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
	_bytes, err := io.ReadAll(&buf)
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
	body, err := io.ReadAll(res.Body)
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
		ErrorMessages: errorMessages, ResponseBodyFunc: func(code int, message string) gin.H {
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
		err := errors.New("")
		AbortWithGError(c, 500, err, "custom hint")
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
		ErrorMessages: errorMessages,
		ErrorStatusFunc: func(code int) bool {
			return code >= 400
		},
	}))
	path := getTestPath()
	router.GET(path, func(c *gin.Context) {
		err := errors.New("error")
		AbortWithGError(c, 400, err, "")
	})
	res := performRequest(router, "GET", path)
	assert.Equal(t, res.Code, 400)
	assert.Equal(t, "error", readLog(t))
}

func TestDefaultResponseFunction(t *testing.T) {
	router := gin.New()
	router.Use(Middleware(MiddlewareOption{ErrorMessages: errorMessages}))

	t.Run("single error", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			err := errors.New("single error")
			AbortWithGError(c, 500, err, "custom hint")
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
			AbortWithGError(c, 500, err, "error1")
			AbortWithGError(c, 400, err, "error2")
		})
		res := performRequest(router, "GET", path)
		// It uses the last error
		assert.Equal(t, res.Code, 400)
		body := parseBody(t, res)
		assert.Equal(t, "error2", body["message"])
	})
	t.Run("only including status code", func(t *testing.T) {
		t.Run("defined status code", func(t *testing.T) {
			path := getTestPath()
			router.GET(path, func(c *gin.Context) {
				AbortWithGError(c, 403, nil, "")
			})
			res := performRequest(router, "GET", path)
			assert.Equal(t, res.Code, 403)
			body := parseBody(t, res)
			assert.Equal(t, "Forbidden", body["message"])
		})
		t.Run("undefined status code", func(t *testing.T) {
			path := getTestPath()
			router.GET(path, func(c *gin.Context) {
				AbortWithGError(c, 402, nil, "")
			})
			res := performRequest(router, "GET", path)
			assert.Equal(t, res.Code, 402)
			_bytes, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Len(t, _bytes, 0)
		})
	})
	t.Run("", func(t *testing.T) {
		path := getTestPath()
		router.GET(path, func(c *gin.Context) {
			AbortWithGError(c, 403, nil, "")
		})
		res := performRequest(router, "GET", path)
		assert.Equal(t, res.Code, 403)
		body := parseBody(t, res)
		assert.Equal(t, "Forbidden", body["message"])
	})
}
