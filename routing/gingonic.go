//go:build gingonic && !gorillamux && !echo
// +build gingonic,!gorillamux,!echo

package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

type ginRouter struct {
	router *gin.Engine
}

func (g ginRouter) Handler() http.Handler {
	return g.router
}

func (g ginRouter) Handle(protocol, route string, handler HandlerFunc) {
	wrappedCallback := func(c *gin.Context) {
		ctx, span := otel.Tracer("").Start(c.Request.Context(), "api2go/gin-handle")
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		params := map[string]string{}
		for _, p := range c.Params {
			params[p.Key] = p.Value
		}

		handler(ctx, c.Writer, c.Request, params, c.Keys)
	}

	g.router.Handle(protocol, route, wrappedCallback)
}

// Gin creates a new api2go router to use with the gin framework
func Gin(g *gin.Engine) Routeable {
	return &ginRouter{router: g}
}
