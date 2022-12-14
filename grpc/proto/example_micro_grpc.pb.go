// Code generated by protoc-gen-go-micro. DO NOT EDIT.
// protoc-gen-go-micro version: v3.5.3
// source: example.proto

package pb

import (
	context "context"
	api "go.unistack.org/micro/v3/api"
	client "go.unistack.org/micro/v3/client"
	server "go.unistack.org/micro/v3/server"
	time "time"
)

type exampleServiceClient struct {
	c    client.Client
	name string
}

func NewExampleServiceClient(name string, c client.Client) ExampleServiceClient {
	return &exampleServiceClient{c: c, name: name}
}

func (c *exampleServiceClient) Hello(ctx context.Context, req *HelloReq, opts ...client.CallOption) (*HelloRsp, error) {
	td := time.Duration(5000000000)
	opts = append(opts, client.WithRequestTimeout(td))
	rsp := &HelloRsp{}
	err := c.c.Call(ctx, c.c.NewRequest(c.name, "ExampleService.Hello", req), rsp, opts...)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

type exampleServiceServer struct {
	ExampleServiceServer
}

func (h *exampleServiceServer) Hello(ctx context.Context, req *HelloReq, rsp *HelloRsp) error {
	var cancel context.CancelFunc
	td := time.Duration(5000000000)
	ctx, cancel = context.WithTimeout(ctx, td)
	defer cancel()
	return h.ExampleServiceServer.Hello(ctx, req, rsp)
}

func RegisterExampleServiceServer(s server.Server, sh ExampleServiceServer, opts ...server.HandlerOption) error {
	type exampleService interface {
		Hello(ctx context.Context, req *HelloReq, rsp *HelloRsp) error
	}
	type ExampleService struct {
		exampleService
	}
	h := &exampleServiceServer{sh}
	var nopts []server.HandlerOption
	for _, endpoint := range ExampleServiceEndpoints {
		nopts = append(nopts, api.WithEndpoint(&endpoint))
	}
	return s.Handle(s.NewHandler(&ExampleService{h}, append(nopts, opts...)...))
}
