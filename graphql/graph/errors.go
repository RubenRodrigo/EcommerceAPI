package graph

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/rasadov/EcommerceAPI/pkg/auth"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const ErrorCodeUnauthenticated = "UNAUTHENTICATED"

func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	presented := graphql.DefaultErrorPresenter(ctx, err)
	if !errors.Is(err, auth.ErrUnauthorized) {
		return presented
	}

	presented.Message = auth.ErrUnauthorized.Error()
	if presented.Extensions == nil {
		presented.Extensions = make(map[string]any)
	}
	presented.Extensions["code"] = ErrorCodeUnauthenticated
	return presented
}
