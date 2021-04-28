gerror
====
Make error handling in Gin easy. It includes a middleware and a few  gin-liked methods to help you to handle errors gracefully for both logging and http response.

[![Build Status](https://travis-ci.com/dcalsky/gerror.svg?branch=master)](https://travis-ci.com/dcalsky/gerror)
[![codecov](https://codecov.io/gh/dcalsky/gerror/branch/master/graph/badge.svg?token=5PLZVKDMVD)](https://codecov.io/gh/dcalsky/gerror)
[![Go Report Card](https://goreportcard.com/badge/github.com/dcalsky/gerror)](https://goreportcard.com/report/github.com/dcalsky/gerror)
[![Go Reference](https://pkg.go.dev/badge/github.com/dcalsky/gerror.svg)](https://pkg.go.dev/github.com/dcalsky/gerror)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/dcalsky/gerror)

# Installation

1. Install package (go version 1.13+ is required):

```shell
$ go get -u github.com/dcalsky/gerror
```

2. Import it in your code:

```go
import "github.com/dcalsky/gerror"
```

# Usage

## Use Middleware with the default options

```go
// Define your default error messages when err message and hint message are empty
errorMessages := map[int]string{
   400: "Bad Request",
   401: "Unauthorized",
   403: "Forbidden",
   404: "Not Found",
   500: "Internal Server Error",
   503: "Service Unavailable",
}
router := gin.New()
router.Use(gerror.Middleware(gerror.MiddlewareOption{ErrorMessages: errorMessages}))
```

Under default option, if an error throwed, the response body (Of course the http header includes: `Content-Type: application/json`) is:

```json
{
   "message": "{your hint message}"
}
```

And if you cause any error, the logrus will log an error level message, like this in the stdOut:

```text
time="2015-03-26T01:27:38-04:00" level=error msg="your error message"
```

### Custom response body

Or you can define your custom response body by passing `ResponseBodyFunc` argument:

```go
router.Use(Middleware(MiddlewareOption{
   ErrorMessages: errorMessages, ResponseBodyFunc: func(code int, message string) gin.H {
      return gin.H{
         "foo":           "foo",
         "code":          code,
         "customMessage": message,
      }
   },
}))
```

### Custom error status validation

By default, gerror middleware only log error whose status code >= 500, but you can define a custom error status function to let it log a wider range of errors.

The gerror middleware can log all errors that has status code >= 400 now with pass `ErrorStatusFunc` argument:

```go
router.Use(Middleware(MiddlewareOption{
   ErrorMessages: errorMessages,
   ErrorStatusFunc: func(code int) bool {
      return code >= 400
   },
}))
```


## Throw the error

```go
router := gin.New()
router.Get("/", func (c *gin.Context) {
  // Your business logics here
  // ... 
  err := gerror.New(500, fmt.Errorf("custom error message: %w", errors.New("error1")), "Error message for user")
  gerror.AbortWithError(c, err)
})
```

Response body will be:

```json
{"message": "Error message for user"}
```

And the logrus will output following error message in stdout:

```
time="2015-03-26T01:27:38-04:00" level=error msg="custom error message: error1"
```

### Throw error in one line

Define a new error then throw it is trivial, so gerror also allow creating and throwing the error in one line:

```go
gerror.AbortWithGError(c, 500, errors.New("custom error"), "Error message for user")
```

### Omit error, only keep hint message

Sometimes, there is no error to throw, leave empty to it:

```go
gerror.AbortWithGError(c, 400, nil, "Bad Input")
```

### Omit hint message and error

```go
gerror.AbortWithGError(c, 404, nil, "") // Equal to: c.AbortWithStatus(404)
```

# Real World

## Example with Gorm

### Base Usage

```go
// ...
db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
foo := model.Foo{
   name: "foo"
}
if err := db.Create(&foo); err != nil {
   gerror.AbortWithError(500, err, "Create Foo instance failed")
   return
}
c.JSON(200, gin.H{
   "foo": &foo,
})
```


### Transaction

```go
// ...
var foo model.Foo
id := 1
if err := r.db.Transaction(func(tx *gorm.DB) error {
   if err := tx.Where("id = ?", id).First(&foo).Error; err == gorm.ErrRecordNotFound {
      return gerror.New(404, nil, "Not found this foo")
   }
   if err := tx.Delete(&foo).Error; err != nil {
      return gerror.New(500, nil, "Delete foo failed, please retry it again")
   }
   return nil
}); err != nil {
   gerror.AbortWithError(c, err)
   return
}
c.JSON(200, gin.H{
   "message": "ok",
})
```