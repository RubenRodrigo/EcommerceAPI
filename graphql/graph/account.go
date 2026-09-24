package graph

import (
	"context"
	"time"

	"github.com/rasadov/EcommerceAPI/graphql/generated"
	"github.com/rasadov/EcommerceAPI/graphql/loaders"
	"github.com/rasadov/EcommerceAPI/graphql/models"
)

type accountResolver struct{}

func (resolver *accountResolver) ID(ctx context.Context, obj *models.Account) (int, error) {
	return int(obj.ID), nil
}

func (resolver *accountResolver) Orders(ctx context.Context, obj *models.Account) ([]*generated.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	requestLoaders, err := loaders.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	orderList, err := requestLoaders.OrdersByAccount.Load(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	orders := make([]*generated.Order, 0, len(orderList))
	for _, order := range orderList {
		var products []*generated.OrderedProduct
		for _, orderedProduct := range order.Products {
			products = append(products, &generated.OrderedProduct{
				ID:          orderedProduct.ID,
				Name:        orderedProduct.Name,
				Description: orderedProduct.Description,
				Price:       orderedProduct.Price,
				Quantity:    int(orderedProduct.Quantity),
			})
		}
		orders = append(orders, &generated.Order{
			ID:         int(order.ID),
			CreatedAt:  order.CreatedAt,
			TotalPrice: order.TotalPrice,
			Products:   products,
		})
	}

	return orders, nil
}
