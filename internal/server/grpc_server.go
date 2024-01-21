package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/status"

	"github.com/nikhilghind/mem-db/internal/engine"
	"github.com/nikhilghind/mem-db/internal/query"
)

func init() {
	encoding.RegisterCodec(JSONCodec{})
}

// JSONCodec is a gRPC codec that uses JSON serialization.
// This avoids the need for protoc-generated code.
type JSONCodec struct{}

func (JSONCodec) Marshal(v interface{}) ([]byte, error)   { return json.Marshal(v) }
func (JSONCodec) Unmarshal(data []byte, v interface{}) error { return json.Unmarshal(data, v) }
func (JSONCodec) Name() string                            { return "proto" }

// ---- Message types matching proto/memdb.proto ----

type InsertRequest struct {
	Table string `json:"table"`
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

type InsertResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GetRequest struct {
	Table string `json:"table"`
	Key   string `json:"key"`
}

type GetResponse struct {
	Found bool   `json:"found"`
	Value []byte `json:"value"`
}

type DeleteRequest struct {
	Table string `json:"table"`
	Key   string `json:"key"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RangeScanRequest struct {
	Table    string `json:"table"`
	StartKey string `json:"start_key"`
	EndKey   string `json:"end_key"`
}

type KeyValueMsg struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Success    bool           `json:"success"`
	Message    string         `json:"message"`
	Value      []byte         `json:"value,omitempty"`
	Rows       []*KeyValueMsg `json:"rows,omitempty"`
	DurationNs int64          `json:"duration_ns"`
}

type StatsRequest struct{}

type TableStatsMsg struct {
	Size   int32 `json:"size"`
	Height int32 `json:"height"`
	Order  int32 `json:"order"`
}

type StatsResponse struct {
	TableCount int32                     `json:"table_count"`
	TotalKeys  int32                     `json:"total_keys"`
	Tables     map[string]*TableStatsMsg `json:"tables"`
}

// ---- gRPC service implementation ----

// MemDBServer implements the MemDB gRPC service.
type MemDBServer struct {
	eng      *engine.Engine
	executor *query.Executor
}

// NewMemDBServer creates a new server backed by the given engine.
func NewMemDBServer(eng *engine.Engine) *MemDBServer {
	return &MemDBServer{
		eng:      eng,
		executor: query.NewExecutor(eng),
	}
}

// Start starts the gRPC server on the given address. Blocks until error or shutdown.
func (s *MemDBServer) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	gs := grpc.NewServer()
	gs.RegisterService(&_MemDB_serviceDesc, s)

	log.Printf("memdb gRPC server listening on %s", addr)
	return gs.Serve(lis)
}

// StartWithServer is like Start but accepts an external *grpc.Server for graceful shutdown.
func (s *MemDBServer) StartWithServer(addr string, gs *grpc.Server) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	gs.RegisterService(&_MemDB_serviceDesc, s)

	log.Printf("memdb gRPC server listening on %s", addr)
	return gs.Serve(lis)
}

// ---- Handlers ----

func (s *MemDBServer) handleInsert(ctx context.Context, req *InsertRequest) (*InsertResponse, error) {
	err := s.eng.Insert(req.Table, req.Key, req.Value)
	if err != nil {
		return &InsertResponse{Success: false, Message: err.Error()}, nil
	}
	return &InsertResponse{Success: true, Message: "OK"}, nil
}

func (s *MemDBServer) handleGet(ctx context.Context, req *GetRequest) (*GetResponse, error) {
	val, found, err := s.eng.Get(req.Table, req.Key)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &GetResponse{Found: found, Value: val}, nil
}

func (s *MemDBServer) handleDelete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	ok, err := s.eng.Delete(req.Table, req.Key)
	if err != nil {
		return &DeleteResponse{Success: false, Message: err.Error()}, nil
	}
	if !ok {
		return &DeleteResponse{Success: false, Message: "key not found"}, nil
	}
	return &DeleteResponse{Success: true, Message: "OK"}, nil
}

func (s *MemDBServer) handleRangeScan(req *RangeScanRequest, stream grpc.ServerStream) error {
	rows, err := s.eng.RangeScan(req.Table, req.StartKey, req.EndKey)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	for _, row := range rows {
		msg := &KeyValueMsg{Key: row.Key, Value: row.Value}
		if err := stream.SendMsg(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemDBServer) handleExecute(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	result := s.executor.ExecuteRaw(req.Query)
	resp := &QueryResponse{
		Success:    result.Success,
		Message:    result.Message,
		Value:      result.Value,
		DurationNs: result.Duration.Nanoseconds(),
	}
	for _, row := range result.Rows {
		resp.Rows = append(resp.Rows, &KeyValueMsg{Key: row.Key, Value: row.Value})
	}
	return resp, nil
}

func (s *MemDBServer) handleStats(ctx context.Context, req *StatsRequest) (*StatsResponse, error) {
	stats := s.eng.Stats()
	resp := &StatsResponse{
		TableCount: int32(stats.TableCount),
		TotalKeys:  int32(stats.TotalKeys),
		Tables:     make(map[string]*TableStatsMsg),
	}
	for name, ts := range stats.Tables {
		resp.Tables[name] = &TableStatsMsg{
			Size:   int32(ts.Size),
			Height: int32(ts.Height),
			Order:  int32(ts.Order),
		}
	}
	return resp, nil
}

// ---- gRPC service descriptor (hand-written, no protoc needed) ----

var _MemDB_serviceDesc = grpc.ServiceDesc{
	ServiceName: "memdb.MemDB",
	HandlerType: (*MemDBServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Insert",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				req := &InsertRequest{}
				if err := dec(req); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*MemDBServer).handleInsert(ctx, req)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/memdb.MemDB/Insert"}
				return interceptor(ctx, req, info, func(ctx context.Context, r interface{}) (interface{}, error) {
					return srv.(*MemDBServer).handleInsert(ctx, r.(*InsertRequest))
				})
			},
		},
		{
			MethodName: "Get",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				req := &GetRequest{}
				if err := dec(req); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*MemDBServer).handleGet(ctx, req)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/memdb.MemDB/Get"}
				return interceptor(ctx, req, info, func(ctx context.Context, r interface{}) (interface{}, error) {
					return srv.(*MemDBServer).handleGet(ctx, r.(*GetRequest))
				})
			},
		},
		{
			MethodName: "Delete",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				req := &DeleteRequest{}
				if err := dec(req); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*MemDBServer).handleDelete(ctx, req)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/memdb.MemDB/Delete"}
				return interceptor(ctx, req, info, func(ctx context.Context, r interface{}) (interface{}, error) {
					return srv.(*MemDBServer).handleDelete(ctx, r.(*DeleteRequest))
				})
			},
		},
		{
			MethodName: "Execute",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				req := &QueryRequest{}
				if err := dec(req); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*MemDBServer).handleExecute(ctx, req)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/memdb.MemDB/Execute"}
				return interceptor(ctx, req, info, func(ctx context.Context, r interface{}) (interface{}, error) {
					return srv.(*MemDBServer).handleExecute(ctx, r.(*QueryRequest))
				})
			},
		},
		{
			MethodName: "Stats",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				req := &StatsRequest{}
				if err := dec(req); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*MemDBServer).handleStats(ctx, req)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/memdb.MemDB/Stats"}
				return interceptor(ctx, req, info, func(ctx context.Context, r interface{}) (interface{}, error) {
					return srv.(*MemDBServer).handleStats(ctx, r.(*StatsRequest))
				})
			},
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName: "RangeScan",
			Handler: func(srv interface{}, stream grpc.ServerStream) error {
				req := &RangeScanRequest{}
				if err := stream.RecvMsg(req); err != nil {
					return err
				}
				return srv.(*MemDBServer).handleRangeScan(req, stream)
			},
			ServerStreams: true,
		},
	},
	Metadata: "proto/memdb.proto",
}
