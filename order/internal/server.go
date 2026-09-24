package internal

import (
	"context"
	"fmt"
	"log"
	"net"

	mapset "github.com/deckarep/golang-set/v2"
	account "github.com/rasadov/EcommerceAPI/account/client"
	"github.com/rasadov/EcommerceAPI/order/models"
	"github.com/rasadov/EcommerceAPI/order/proto/pb"
	product "github.com/rasadov/EcommerceAPI/product/client"
	productModels "github.com/rasadov/EcommerceAPI/product/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type grpcServer struct {
	pb.UnimplementedOrderServiceServer
	service       Service
	accountClient *account.Client
	productClient *product.Client
}

func ListenGRPC(service Service, accountURL string, productURL string, port int) error {
	accountClient, err := account.NewClient(accountURL)
	if err != nil {
		return err
	}

	productClient, err := product.NewClient(productURL)
	if err != nil {
		accountClient.Close()
		return err
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		accountClient.Close()
		productClient.Close()
		return err
	}

	serv := grpc.NewServer()
	pb.RegisterOrderServiceServer(serv, &grpcServer{
		pb.UnimplementedOrderServiceServer{},
		service,
		accountClient,
		productClient,
	})
	reflection.Register(serv)

	return serv.Serve(lis)
}

func (server *grpcServer) PostOrder(ctx context.Context, request *pb.PostOrderRequest) (*pb.PostOrderResponse, error) {
	_, err := server.accountClient.GetAccount(ctx, request.AccountId)
	if err != nil {
		log.Println("Error getting account", err)
		return nil, err
	}
	var productIDs []string
	for _, p := range request.Products {
		productIDs = append(productIDs, p.Id)
	}
	orderedProducts, err := server.productClient.GetProducts(ctx, 0, 0, productIDs, "")
	if err != nil {
		log.Println("Error getting ordered products", err)
		return nil, err
	}

	var products []*models.OrderedProduct
	totalPrice := 0.0

	for _, p := range orderedProducts {
		productObj := &models.OrderedProduct{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Quantity:    0,
		}
		for _, requestProduct := range request.Products {
			if requestProduct.Id == p.ID {
				productObj.Quantity = requestProduct.Quantity
				break
			}
		}

		if productObj.Quantity != 0 {
			products = append(products, productObj)
			totalPrice += productObj.Price * float64(productObj.Quantity)
		}
	}

	postOrder, err := server.service.PostOrder(ctx, request.AccountId, totalPrice, products)
	if err != nil {
		log.Println("Error posting postOrder", err)
		return nil, err
	}

	orderProto := &pb.Order{
		Id:         uint64(postOrder.ID),
		AccountId:  postOrder.AccountID,
		TotalPrice: postOrder.TotalPrice,
		Products:   []*pb.ProductInfo{},
	}
	orderProto.CreatedAt, _ = postOrder.CreatedAt.MarshalBinary()
	for _, p := range postOrder.Products {
		orderProto.Products = append(orderProto.Products, &pb.ProductInfo{
			Id:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Quantity:    p.Quantity,
		})
	}

	return &pb.PostOrderResponse{
		Order: orderProto,
	}, nil
}

func (server *grpcServer) GetOrdersForAccount(ctx context.Context, request *wrapperspb.UInt64Value) (*pb.GetOrdersForAccountResponse, error) {
	ordersByAccount, err := server.getOrdersForAccounts(ctx, []uint64{request.Value})
	if err != nil {
		return nil, err
	}
	return &pb.GetOrdersForAccountResponse{Orders: ordersByAccount[request.Value]}, nil
}

func (server *grpcServer) GetOrdersForAccounts(ctx context.Context, request *pb.GetOrdersForAccountsRequest) (*pb.GetOrdersForAccountsResponse, error) {
	ordersByAccount, err := server.getOrdersForAccounts(ctx, request.AccountIds)
	if err != nil {
		return nil, err
	}

	response := &pb.GetOrdersForAccountsResponse{
		Accounts: make([]*pb.AccountOrders, 0, len(request.AccountIds)),
	}
	for _, accountID := range request.AccountIds {
		response.Accounts = append(response.Accounts, &pb.AccountOrders{
			AccountId: accountID,
			Orders:    ordersByAccount[accountID],
		})
	}
	return response, nil
}

func (server *grpcServer) getOrdersForAccounts(ctx context.Context, accountIDs []uint64) (map[uint64][]*pb.Order, error) {
	accountOrders, err := server.service.GetOrdersForAccounts(ctx, accountIDs)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// Taking unique products. We use set to avoid repeating
	productIDsSet := mapset.NewSet[string]()
	for _, orders := range accountOrders {
		for _, order := range orders {
			for _, product := range order.Products {
				productIDsSet.Add(product.ID)
			}
		}
	}

	productIDs := productIDsSet.ToSlice()
	productsByID := make(map[string]productModels.Product, len(productIDs))
	if len(productIDs) > 0 {
		products, err := server.productClient.GetProducts(ctx, 0, 0, productIDs, "")
		if err != nil {
			log.Println("Error getting account products: ", err)
			return nil, err
		}
		for _, product := range products {
			productsByID[product.ID] = product
		}
	}

	result := make(map[uint64][]*pb.Order, len(accountIDs))
	for _, accountID := range accountIDs {
		result[accountID] = []*pb.Order{}
		for _, order := range accountOrders[accountID] {
			encodedOrder := &pb.Order{
				AccountId:  order.AccountID,
				Id:         uint64(order.ID),
				TotalPrice: order.TotalPrice,
				Products:   []*pb.ProductInfo{},
			}
			encodedOrder.CreatedAt, _ = order.CreatedAt.MarshalBinary()

			for _, orderedProduct := range order.Products {
				product := productsByID[orderedProduct.ID]
				encodedOrder.Products = append(encodedOrder.Products, &pb.ProductInfo{
					Id:          orderedProduct.ID,
					Name:        product.Name,
					Description: product.Description,
					Price:       product.Price,
					Quantity:    orderedProduct.Quantity,
				})
			}
			result[accountID] = append(result[accountID], encodedOrder)
		}
	}
	return result, nil
}

func (server *grpcServer) UpdateOrderStatus(ctx context.Context, request *pb.UpdateOrderStatusRequest) (*emptypb.Empty, error) {
	err := server.service.UpdateOrderPaymentStatus(ctx, request.OrderId, request.Status)

	if err != nil {
		log.Println("Error updating order status", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
