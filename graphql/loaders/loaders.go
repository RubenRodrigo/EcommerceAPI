package loaders

import (
	"context"
	"errors"
	"time"

	"github.com/vikstrous/dataloadgen"

	orderModels "github.com/rasadov/EcommerceAPI/order/models"
)

const (
	orderLoaderWait          = 2 * time.Millisecond
	orderLoaderBatchCapacity = 100
)

type OrdersClient interface {
	GetOrdersForAccounts(ctx context.Context, accountIDs []uint64) (map[uint64][]orderModels.Order, error)
}

type Loaders struct {
	OrdersByAccount *dataloadgen.Loader[uint64, []orderModels.Order]
}

func New(orderClient OrdersClient) *Loaders {
	return &Loaders{
		OrdersByAccount: dataloadgen.NewMappedLoader(
			orderClient.GetOrdersForAccounts,
			dataloadgen.WithWait(orderLoaderWait),
			dataloadgen.WithBatchCapacity(orderLoaderBatchCapacity),
		),
	}
}

type contextKey struct{}

func WithContext(ctx context.Context, requestLoaders *Loaders) context.Context {
	return context.WithValue(ctx, contextKey{}, requestLoaders)
}

func FromContext(ctx context.Context) (*Loaders, error) {
	requestLoaders, ok := ctx.Value(contextKey{}).(*Loaders)
	if !ok || requestLoaders == nil {
		return nil, errors.New("dataloaders are missing from request context")
	}
	return requestLoaders, nil
}
