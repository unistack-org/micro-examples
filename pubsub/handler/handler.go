package handler

import (
	"context"

	"github.com/jmoiron/sqlx"
	pb "github.com/unistack-org/micro-examples/pubsub/proto"
)

type Handler struct {
	db *sqlx.DB
}

func NewHandler(db *sqlx.DB) (*Handler, error) {
	return &Handler{
		db: db,
	}, nil
}

func (h *Handler) Hello(ctx context.Context, req *pb.HelloReq, rsp *pb.HelloRsp) error {
	return nil
}
