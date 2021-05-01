package gerror

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type GError struct {
	Code int    `json:"code"`
	Err  error  `json:"err"`
	Hint string `json:"hint"`
}

func (g GError) Error() string {
	if g.Err == nil {
		return ""
	}
	return g.Err.Error()
}

func New(code int, err error, hint string) error {
	return GError{
		code,
		err,
		hint,
	}
}

func NewHint(code int, hint string) error {
	return New(code, nil, hint)
}

func NewEmpty(code int) error {
	return New(code, nil, "")
}

func AbortWithHint(c *gin.Context, code int, hint string) {
	AbortWithErrorAndHint(c, code, nil, hint)
}

func AbortWithErrorAndHint(c *gin.Context, code int, err error, hint string) {
	if _, ok := err.(GError); !ok {
		err = New(code, err, hint)
	}
	c.Abort()
	c.Errors = append(c.Errors, &gin.Error{
		Err:  err,
		Type: gin.ErrorTypePrivate,
	})
}

func AbortWithError(c *gin.Context, code int, err error) {
	AbortWithErrorAndHint(c, code, err, "")
}

type MiddlewareOption struct {
	ResponseBodyFunc func(code int, message string) interface{}
	LoggingFunc      func(code int, err error)
}

func Middleware(option MiddlewareOption) gin.HandlerFunc {
	if option.ResponseBodyFunc == nil {
		option.ResponseBodyFunc = func(code int, message string) interface{} {
			if message == "" {
				return nil
			}
			return gin.H{
				"message": message,
			}
		}
	}
	if option.LoggingFunc == nil {
		option.LoggingFunc = func(code int, err error) {
			if code >= 500 {
				logrus.Errorln(err)
			}
		}
	}
	return func(c *gin.Context) {
		c.Next()
		lastError := c.Errors.Last()
		if c.IsAborted() && lastError != nil {
			var message string
			code := c.Writer.Status()
			if gError, ok := lastError.Err.(GError); ok {
				code = gError.Code
				message = gError.Hint
			} else {
				message = lastError.Error()
			}
			option.LoggingFunc(code, lastError)
			body := option.ResponseBodyFunc(code, message)
			if body == nil {
				c.Status(code)
			} else {
				c.JSON(code, body)
			}
		}
	}
}
