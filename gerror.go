package gerror

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type GError struct {
	Code int    `json:"code"`
	Err  error  `json:"err"`
	Hint string `json:"hint"`
}

func (g GError) Error() string {
	return g.Err.Error()
}

func New(code int, err error, hint string) error {
	var e error
	if err != nil {
		e = err
	} else {
		e = errors.New("")
	}
	return GError{
		code,
		e,
		hint,
	}
}

func AbortWithGError(c *gin.Context, code int, err error, hint string) {
	AbortWithError(c, New(code, err, hint))
}

func AbortWithError(c *gin.Context, err error) {
	c.Abort()
	c.Errors = append(c.Errors, &gin.Error{
		Err:  err,
		Type: gin.ErrorTypePrivate,
	})
}

type MiddlewareOption struct {
	ResponseBodyFunc func(code int, message string) gin.H
	ErrorStatusFunc  func(code int) bool
	ErrorMessages    map[int]string
}

func Middleware(option MiddlewareOption) gin.HandlerFunc {
	if option.ResponseBodyFunc == nil {
		option.ResponseBodyFunc = func(code int, message string) gin.H {
			return gin.H{
				"message": message,
			}
		}
	}
	if option.ErrorStatusFunc == nil {
		option.ErrorStatusFunc = func(code int) bool {
			return code >= 500
		}
	}
	return func(c *gin.Context) {
		c.Next()
		lastError := c.Errors.Last()
		if c.IsAborted() {
			var message string
			code := c.Writer.Status()
			if lastError == nil {
				return
			}
			if gError, ok := lastError.Err.(GError); ok {
				if gError.Code >= 200 && gError.Code < 600 {
					code = gError.Code
				}
				if gError.Hint != "" {
					message = gError.Hint
				}
			} else {
				message = lastError.Error()
			}
			if option.ErrorStatusFunc(code) {
				logrus.Errorln(lastError)
			}
			if message == "" {
				if option.ErrorMessages[code] != "" {
					message = option.ErrorMessages[code]
					c.JSON(code, option.ResponseBodyFunc(code, message))
				} else {
					c.Status(code)
				}
			} else {
				c.JSON(code, option.ResponseBodyFunc(code, message))
			}
		}
	}
}
