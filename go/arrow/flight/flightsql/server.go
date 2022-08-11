// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flightsql

import (
	"context"
	"fmt"

	"github.com/apache/arrow/go/v10/arrow"
	"github.com/apache/arrow/go/v10/arrow/array"
	"github.com/apache/arrow/go/v10/arrow/flight"
	"github.com/apache/arrow/go/v10/arrow/flight/flightsql/schema_ref"
	pb "github.com/apache/arrow/go/v10/arrow/flight/internal/flight"
	"github.com/apache/arrow/go/v10/arrow/internal/debug"
	"github.com/apache/arrow/go/v10/arrow/ipc"
	"github.com/apache/arrow/go/v10/arrow/memory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// the following interfaces wrap the Protobuf commands to avoid
// exposing the Protobuf types themselves in the API.

// StatementQuery represents a Sql Query
type StatementQuery interface {
	GetQuery() string
}

// StatementUpdate represents a SQL update query
type StatementUpdate interface {
	GetQuery() string
}

// StatementQueryTicket represents a request to execute a query
type StatementQueryTicket interface {
	// GetStatementHandle returns the server-generated opaque
	// identifier for the query
	GetStatementHandle() []byte
}

// PreparedStatementQuery represents a prepared query statement
type PreparedStatementQuery interface {
	// GetPreparedStatementHandle returns the server-generated opaque
	// identifier for the statement
	GetPreparedStatementHandle() []byte
}

// PreparedStatementUpdate represents a prepared update statement
type PreparedStatementUpdate interface {
	// GetPreparedStatementHandle returns the server-generated opaque
	// identifier for the statement
	GetPreparedStatementHandle() []byte
}

// ActionClosePreparedStatementRequest represents a request to close
// a prepared statement
type ActionClosePreparedStatementRequest interface {
	// GetPreparedStatementHandle returns the server-generated opaque
	// identifier for the statement
	GetPreparedStatementHandle() []byte
}

// ActionCreatePreparedStatementRequest represents a request to construct
// a new prepared statement
type ActionCreatePreparedStatementRequest interface {
	GetQuery() string
}

// ActionCreatePreparedStatementResult is the result of creating a new
// prepared statement, optionally including the dataset and parameter
// schemas.
type ActionCreatePreparedStatementResult struct {
	Handle          []byte
	DatasetSchema   *arrow.Schema
	ParameterSchema *arrow.Schema
}

type getXdbcTypeInfo struct {
	*pb.CommandGetXdbcTypeInfo
}

func (c *getXdbcTypeInfo) GetDataType() *int32 { return c.DataType }

// GetXdbcTypeInfo represents a request for SQL Data Type information
type GetXdbcTypeInfo interface {
	// GetDataType returns either nil (get for all types)
	// or a specific SQL type ID to fetch information about.
	GetDataType() *int32
}

// GetSqlInfo represents a request for SQL Information
type GetSqlInfo interface {
	// GetInfo returns a slice of SqlInfo ids to return information about
	GetInfo() []uint32
}

type getDBSchemas struct {
	*pb.CommandGetDbSchemas
}

func (c *getDBSchemas) GetCatalog() *string               { return c.Catalog }
func (c *getDBSchemas) GetDBSchemaFilterPattern() *string { return c.DbSchemaFilterPattern }

// GetDBSchemas represents a request for list of database schemas
type GetDBSchemas interface {
	GetCatalog() *string
	GetDBSchemaFilterPattern() *string
}

type getTables struct {
	*pb.CommandGetTables
}

func (c *getTables) GetCatalog() *string                { return c.Catalog }
func (c *getTables) GetDBSchemaFilterPattern() *string  { return c.DbSchemaFilterPattern }
func (c *getTables) GetTableNameFilterPattern() *string { return c.TableNameFilterPattern }

// GetTables represents a request to list the database's tables
type GetTables interface {
	GetCatalog() *string
	GetDBSchemaFilterPattern() *string
	GetTableNameFilterPattern() *string
	GetTableTypes() []string
	GetIncludeSchema() bool
}

// BaseServer must be embedded into any FlightSQL Server implementation
// and provides default implementations of all methods returning an
// unimplemented error if called. This allows consumers to gradually
// implement methods as they want instead of requiring all consumers to
// boilerplate the same "unimplemented" methods.
//
// The base implementation also contains handling for registering sql info
// and serving it up in response to GetSqlInfo requests.
type BaseServer struct {
	sqlInfoToResult SqlInfoResultMap
	// Alloc allows specifying a particular allocator to use for any
	// allocations done by the base implementation.
	// Will use memory.DefaultAlloctor if nil
	Alloc memory.Allocator
}

func (BaseServer) mustEmbedBaseServer() {}

// RegisterSqlInfo registers a specific result to return for a given sqlinfo
// id. The result must be one of the following types: string, bool, int64,
// int32, []string, or map[int32][]int32.
//
// Once registered, this value will be returned for any SqlInfo requests.
func (b *BaseServer) RegisterSqlInfo(id SqlInfo, result interface{}) error {
	switch result.(type) {
	case string, bool, int64, int32, []string, map[int32][]int32:
		b.sqlInfoToResult[uint32(id)] = result
	default:
		return fmt.Errorf("invalid sql info type '%T' registered for id: %d", result, id)
	}
	return nil
}

func (BaseServer) GetFlightInfoStatement(context.Context, StatementQuery, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetFlightInfoStatement not implemented")
}

func (BaseServer) DoGetStatement(context.Context, StatementQueryTicket) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetStatement not implemented")
}

func (BaseServer) GetFlightInfoPreparedStatement(context.Context, PreparedStatementQuery, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetFlightInfoPreparedStatement not implemented")
}

func (BaseServer) DoGetPreparedStatement(context.Context, PreparedStatementQuery) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetPreparedStatement not implemented")
}

func (BaseServer) GetFlightInfoCatalogs(context.Context, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetFlightInfoCatalogs not implemented")
}

func (BaseServer) DoGetCatalogs(context.Context) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetCatalogs not implemented")
}

func (BaseServer) GetFlightInfoXdbcTypeInfo(context.Context, GetXdbcTypeInfo, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetFlightInfoXdbcTypeInfo not implemented")
}

func (BaseServer) DoGetXdbcTypeInfo(context.Context, GetXdbcTypeInfo) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetXdbcTypeInfo not implemented")
}

// GetFlightInfoSqlInfo is a base implementation of GetSqlInfo by using any
// registered sqlinfo (by calling RegisterSqlInfo). Will return an error
// if there is no sql info registered, otherwise a FlightInfo for retrieving
// the Sql info.
func (b *BaseServer) GetFlightInfoSqlInfo(_ context.Context, _ GetSqlInfo, desc *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	if len(b.sqlInfoToResult) == 0 {
		return nil, status.Error(codes.NotFound, "no sql information available")
	}

	if b.Alloc == nil {
		b.Alloc = memory.DefaultAllocator
	}

	return &flight.FlightInfo{
		Endpoint:         []*flight.FlightEndpoint{{Ticket: &flight.Ticket{Ticket: desc.Cmd}}},
		FlightDescriptor: desc,
		TotalRecords:     -1,
		TotalBytes:       -1,
		Schema:           flight.SerializeSchema(schema_ref.SqlInfo, b.Alloc),
	}, nil
}

// DoGetSqlInfo returns a flight stream containing the list of sqlinfo results
func (b *BaseServer) DoGetSqlInfo(_ context.Context, cmd GetSqlInfo) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	if b.Alloc == nil {
		b.Alloc = memory.DefaultAllocator
	}

	bldr := array.NewRecordBuilder(b.Alloc, schema_ref.SqlInfo)
	defer bldr.Release()

	nameFieldBldr := bldr.Field(0).(*array.Uint32Builder)
	valFieldBldr := bldr.Field(1).(*array.DenseUnionBuilder)

	// doesn't take ownership, no calls to retain. so we don't need
	// extra releases.
	sqlInfoResultBldr := newSqlInfoResultBuilder(valFieldBldr)

	// populate both the nameFieldBldr and the values for each
	// element on command.info.
	// valueFieldBldr is populated depending on the data type
	// since it's a dense union. The population for each
	// data type is handled by the sqlInfoResultBuilder.
	for _, info := range cmd.GetInfo() {
		val, ok := b.sqlInfoToResult[info]
		if !ok {
			return nil, nil, status.Errorf(codes.NotFound, "no information for sql info number %d", info)
		}
		nameFieldBldr.Append(info)
		sqlInfoResultBldr.Append(val)
	}

	batch := bldr.NewRecord()
	defer batch.Release()
	debug.Assert(int(batch.NumRows()) == len(cmd.GetInfo()), "too many rows added to SqlInfo result")

	ch := make(chan flight.StreamChunk)
	rdr, err := array.NewRecordReader(schema_ref.SqlInfo, []arrow.Record{batch})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "error producing record response: %s", err.Error())
	}

	// StreamChunksFromReader will call release on the reader when done
	go flight.StreamChunksFromReader(rdr, ch)
	return schema_ref.SqlInfo, ch, nil
}

func (BaseServer) GetFlightInfoSchemas(context.Context, GetDBSchemas, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetFlightInfoSchemas not implemented")
}

func (BaseServer) DoGetDBSchemas(context.Context, GetDBSchemas) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetDBSchemas not implemented")
}

func (BaseServer) GetFlightInfoTables(context.Context, GetTables, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetFlightInfoTables not implemented")
}

func (BaseServer) DoGetTables(context.Context, GetTables) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetTables not implemented")
}

func (BaseServer) GetFlightInfoTableTypes(context.Context, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetFlightInfoTableTypes not implemented")
}

func (BaseServer) DoGetTableTypes(context.Context) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetTableTypes not implemented")
}

func (BaseServer) GetFlightInfoPrimaryKeys(context.Context, TableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Error(codes.Unimplemented, "GetFlightInfoPrimaryKeys not implemented")
}

func (BaseServer) DoGetPrimaryKeys(context.Context, TableRef) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetPrimaryKeys not implemented")
}

func (BaseServer) GetFlightInfoExportedKeys(context.Context, TableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Error(codes.Unimplemented, "GetFlightInfoExportedKeys not implemented")
}

func (BaseServer) DoGetExportedKeys(context.Context, TableRef) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetExportedKeys not implemented")
}

func (BaseServer) GetFlightInfoImportedKeys(context.Context, TableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Error(codes.Unimplemented, "GetFlightInfoImportedKeys not implemented")
}

func (BaseServer) DoGetImportedKeys(context.Context, TableRef) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetImportedKeys not implemented")
}

func (BaseServer) GetFlightInfoCrossReference(context.Context, CrossTableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	return nil, status.Error(codes.Unimplemented, "GetFlightInfoCrossReference not implemented")
}

func (BaseServer) DoGetCrossReference(context.Context, CrossTableRef) (*arrow.Schema, <-chan flight.StreamChunk, error) {
	return nil, nil, status.Errorf(codes.Unimplemented, "DoGetCrossReference not implemented")
}

func (BaseServer) CreatePreparedStatement(context.Context, ActionCreatePreparedStatementRequest) (res ActionCreatePreparedStatementResult, err error) {
	return res, status.Error(codes.Unimplemented, "CreatePreparedStatement not implemented")
}

func (BaseServer) ClosePreparedStatement(context.Context, ActionClosePreparedStatementRequest) error {
	return status.Error(codes.Unimplemented, "ClosePreparedStatement not implemented")
}

func (BaseServer) DoPutCommandStatementUpdate(context.Context, StatementUpdate) (int64, error) {
	return 0, status.Error(codes.Unimplemented, "DoPutCommandStatementUpdate not implemented")
}
func (BaseServer) DoPutPreparedStatementQuery(context.Context, PreparedStatementQuery, flight.MessageReader, flight.MetadataWriter) error {
	return status.Error(codes.Unimplemented, "DoPutPreparedStatementQuery not implemented")
}

func (BaseServer) DoPutPreparedStatementUpdate(context.Context, PreparedStatementUpdate, flight.MessageReader) (int64, error) {
	return 0, status.Error(codes.Unimplemented, "DoPutPreparedStatementUpdate not implemented")
}

// Server is the required interface for a FlightSQL server. It is implemented by
// BaseServer which must be embedded in any implementation. The default
// implementation by BaseServer for each of these (except GetSqlInfo)
//
// GetFlightInfo* methods should return the FlightInfo object representing where
// to retrieve the results for a given request.
//
// DoGet* methods should return the Schema of the resulting stream along with
// a channel to retrieve stream chunks (each chunk is a record batch and optionally
// a descriptor and app metadata). The channel will be read from until it
// closes, sending each chunk on the stream. Since the channel is returned
// from the method, it should be populated within a goroutine to ensure
// there are no deadlocks.
type Server interface {
	// GetFlightInfoStatement returns a FlightInfo for executing the requested sql query
	GetFlightInfoStatement(context.Context, StatementQuery, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetStatement returns a stream containing the query results for the
	// requested statement handle that was populated by GetFlightInfoStatement
	DoGetStatement(context.Context, StatementQueryTicket) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoPreparedStatement returns a FlightInfo for executing an already
	// prepared statement with the provided statement handle.
	GetFlightInfoPreparedStatement(context.Context, PreparedStatementQuery, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetPreparedStatement returns a stream containing the results from executing
	// a prepared statement query with the provided statement handle.
	DoGetPreparedStatement(context.Context, PreparedStatementQuery) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoCatalogs returns a FlightInfo for the listing of all catalogs
	GetFlightInfoCatalogs(context.Context, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetCatalogs returns the stream containing the list of catalogs
	DoGetCatalogs(context.Context) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoXdbcTypeInfo returns a FlightInfo for retrieving data type info
	GetFlightInfoXdbcTypeInfo(context.Context, GetXdbcTypeInfo, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetXdbcTypeInfo returns a stream containing the information about the
	// requested supported datatypes
	DoGetXdbcTypeInfo(context.Context, GetXdbcTypeInfo) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoSqlInfo returns a FlightInfo for retrieving SqlInfo from the server
	GetFlightInfoSqlInfo(context.Context, GetSqlInfo, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetSqlInfo returns a stream containing the list of SqlInfo results
	DoGetSqlInfo(context.Context, GetSqlInfo) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoSchemas returns a FlightInfo for requesting a list of schemas
	GetFlightInfoSchemas(context.Context, GetDBSchemas, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetDBSchemas returns a stream containing the list of schemas
	DoGetDBSchemas(context.Context, GetDBSchemas) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoTables returns a FlightInfo for listing the tables available
	GetFlightInfoTables(context.Context, GetTables, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetTables returns a stream containing the list of tables
	DoGetTables(context.Context, GetTables) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoTableTypes returns a FlightInfo for retrieving a list
	// of table types supported
	GetFlightInfoTableTypes(context.Context, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetTableTypes returns a stream containing the data related to the table types
	DoGetTableTypes(context.Context) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoPrimaryKeys returns a FlightInfo for extracting information about primary keys
	GetFlightInfoPrimaryKeys(context.Context, TableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetPrimaryKeys returns a stream containing the data related to primary keys
	DoGetPrimaryKeys(context.Context, TableRef) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoExportedKeys returns a FlightInfo for extracting information about foreign keys
	GetFlightInfoExportedKeys(context.Context, TableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetExportedKeys returns a stream containing the data related to foreign keys
	DoGetExportedKeys(context.Context, TableRef) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoImportedKeys returns a FlightInfo for extracting information about imported keys
	GetFlightInfoImportedKeys(context.Context, TableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetImportedKeys returns a stream containing the data related to imported keys
	DoGetImportedKeys(context.Context, TableRef) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// GetFlightInfoCrossReference returns a FlightInfo for extracting data related
	// to primary and foreign keys
	GetFlightInfoCrossReference(context.Context, CrossTableRef, *flight.FlightDescriptor) (*flight.FlightInfo, error)
	// DoGetCrossReference returns a stream of data related to foreign and primary keys
	DoGetCrossReference(context.Context, CrossTableRef) (*arrow.Schema, <-chan flight.StreamChunk, error)
	// DoPutCommandStatementUpdate executes a sql update statement and returns
	// the number of affected rows
	DoPutCommandStatementUpdate(context.Context, StatementUpdate) (int64, error)
	// CreatePreparedStatement constructs a prepared statement from a sql query
	// and returns an opaque statement handle for use.
	CreatePreparedStatement(context.Context, ActionCreatePreparedStatementRequest) (ActionCreatePreparedStatementResult, error)
	// ClosePreparedStatement closes the prepared statement identified by the requested
	// opaque statement handle.
	ClosePreparedStatement(context.Context, ActionClosePreparedStatementRequest) error
	// DoPutPreparedStatementQuery binds parameters to a given prepared statement
	// identified by the provided statement handle.
	//
	// The provided MessageReader is a stream of record batches with optional
	// app metadata and flight descriptors to represent the values to bind
	// to the parameters.
	//
	// Currently anything written to the writer will be ignored. It is in the
	// interface for potential future enhancements to avoid having to change
	// the interface in the future.
	DoPutPreparedStatementQuery(context.Context, PreparedStatementQuery, flight.MessageReader, flight.MetadataWriter) error
	// DoPutPreparedStatementUpdate executes an update SQL Prepared statement
	// for the specified statement handle. The reader allows providing a sequence
	// of uploaded record batches to bind the parameters to. Returns the number
	// of affected records.
	DoPutPreparedStatementUpdate(context.Context, PreparedStatementUpdate, flight.MessageReader) (int64, error)

	mustEmbedBaseServer()
}

// NewFlightServer constructs a FlightRPC server from the provided
// FlightSQL Server so that it can be passed to RegisterFlightService.
func NewFlightServer(srv Server) flight.FlightServer {
	return &flightSqlServer{srv: srv, mem: memory.DefaultAllocator}
}

// NewFlightServerWithAllocator constructs a FlightRPC server from
// the provided FlightSQL Server so that it can be passed to
// RegisterFlightService, setting the provided allocator into the server
// for use with any allocations necessary by the routing.
//
// Will default to memory.DefaultAllocator if mem is nil
func NewFlightServerWithAllocator(srv Server, mem memory.Allocator) flight.FlightServer {
	if mem == nil {
		mem = memory.DefaultAllocator
	}
	return &flightSqlServer{srv: srv, mem: mem}
}

// flightSqlServer is a wrapper around a FlightSQL server interface to
// perform routing from FlightRPC to FlightSQL.
type flightSqlServer struct {
	flight.BaseFlightServer
	mem memory.Allocator
	srv Server
}

func (f *flightSqlServer) GetFlightInfo(ctx context.Context, request *flight.FlightDescriptor) (*flight.FlightInfo, error) {
	var (
		anycmd anypb.Any
		cmd    proto.Message
		err    error
	)
	if err = proto.Unmarshal(request.Cmd, &anycmd); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to parse command: %s", err.Error())
	}

	if cmd, err = anycmd.UnmarshalNew(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not unmarshal Any to a command type: %s", err.Error())
	}

	switch cmd := cmd.(type) {
	case *pb.CommandStatementQuery:
		return f.srv.GetFlightInfoStatement(ctx, cmd, request)
	case *pb.CommandPreparedStatementQuery:
		return f.srv.GetFlightInfoPreparedStatement(ctx, cmd, request)
	case *pb.CommandGetCatalogs:
		return f.srv.GetFlightInfoCatalogs(ctx, request)
	case *pb.CommandGetDbSchemas:
		return f.srv.GetFlightInfoSchemas(ctx, &getDBSchemas{cmd}, request)
	case *pb.CommandGetTables:
		return f.srv.GetFlightInfoTables(ctx, &getTables{cmd}, request)
	case *pb.CommandGetTableTypes:
		return f.srv.GetFlightInfoTableTypes(ctx, request)
	case *pb.CommandGetXdbcTypeInfo:
		return f.srv.GetFlightInfoXdbcTypeInfo(ctx, &getXdbcTypeInfo{cmd}, request)
	case *pb.CommandGetSqlInfo:
		return f.srv.GetFlightInfoSqlInfo(ctx, cmd, request)
	case *pb.CommandGetPrimaryKeys:
		return f.srv.GetFlightInfoPrimaryKeys(ctx, pkToTableRef(cmd), request)
	case *pb.CommandGetExportedKeys:
		return f.srv.GetFlightInfoExportedKeys(ctx, exkToTableRef(cmd), request)
	case *pb.CommandGetImportedKeys:
		return f.srv.GetFlightInfoImportedKeys(ctx, impkToTableRef(cmd), request)
	case *pb.CommandGetCrossReference:
		return f.srv.GetFlightInfoCrossReference(ctx, toCrossTableRef(cmd), request)
	}

	return nil, status.Error(codes.InvalidArgument, "requested command is invalid")
}

func (f *flightSqlServer) DoGet(request *flight.Ticket, stream flight.FlightService_DoGetServer) (err error) {
	var (
		anycmd anypb.Any
		cmd    proto.Message
		cc     <-chan flight.StreamChunk
		sc     *arrow.Schema
	)
	if err = proto.Unmarshal(request.Ticket, &anycmd); err != nil {
		return status.Errorf(codes.InvalidArgument, "unable to parse ticket: %s", err.Error())
	}

	if cmd, err = anycmd.UnmarshalNew(); err != nil {
		return status.Errorf(codes.InvalidArgument, "unable to unmarshal proto.Any: %s", err.Error())
	}

	switch cmd := cmd.(type) {
	case *pb.TicketStatementQuery:
		sc, cc, err = f.srv.DoGetStatement(stream.Context(), cmd)
	case *pb.CommandPreparedStatementQuery:
		sc, cc, err = f.srv.DoGetPreparedStatement(stream.Context(), cmd)
	case *pb.CommandGetCatalogs:
		sc, cc, err = f.srv.DoGetCatalogs(stream.Context())
	case *pb.CommandGetDbSchemas:
		sc, cc, err = f.srv.DoGetDBSchemas(stream.Context(), &getDBSchemas{cmd})
	case *pb.CommandGetTables:
		sc, cc, err = f.srv.DoGetTables(stream.Context(), &getTables{cmd})
	case *pb.CommandGetTableTypes:
		sc, cc, err = f.srv.DoGetTableTypes(stream.Context())
	case *pb.CommandGetXdbcTypeInfo:
		sc, cc, err = f.srv.DoGetXdbcTypeInfo(stream.Context(), &getXdbcTypeInfo{cmd})
	case *pb.CommandGetSqlInfo:
		sc, cc, err = f.srv.DoGetSqlInfo(stream.Context(), cmd)
	case *pb.CommandGetPrimaryKeys:
		sc, cc, err = f.srv.DoGetPrimaryKeys(stream.Context(), pkToTableRef(cmd))
	case *pb.CommandGetExportedKeys:
		sc, cc, err = f.srv.DoGetExportedKeys(stream.Context(), exkToTableRef(cmd))
	case *pb.CommandGetImportedKeys:
		sc, cc, err = f.srv.DoGetImportedKeys(stream.Context(), impkToTableRef(cmd))
	case *pb.CommandGetCrossReference:
		sc, cc, err = f.srv.DoGetCrossReference(stream.Context(), toCrossTableRef(cmd))
	default:
		return status.Error(codes.InvalidArgument, "requested command is invalid")
	}

	if err != nil {
		return err
	}

	wr := flight.NewRecordWriter(stream, ipc.WithSchema(sc))
	defer wr.Close()

	for chunk := range cc {
		if chunk.Err != nil {
			return err
		}

		wr.SetFlightDescriptor(chunk.Desc)
		if err = wr.WriteWithAppMetadata(chunk.Data, chunk.AppMetadata); err != nil {
			return err
		}
		chunk.Data.Release()
	}

	return err
}

type putMetadataWriter struct {
	stream flight.FlightService_DoPutServer
}

func (p *putMetadataWriter) WriteMetadata(appMetadata []byte) error {
	return p.stream.Send(&flight.PutResult{AppMetadata: appMetadata})
}

func (f *flightSqlServer) DoPut(stream flight.FlightService_DoPutServer) error {
	rdr, err := flight.NewRecordReader(stream, ipc.WithAllocator(f.mem), ipc.WithDelayReadSchema(true))
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to read input stream: %s", err.Error())
	}
	defer rdr.Release()

	// flight descriptor should have come with the schema message
	request := rdr.LatestFlightDescriptor()

	var (
		anycmd anypb.Any
		cmd    proto.Message
	)
	if err = proto.Unmarshal(request.Cmd, &anycmd); err != nil {
		return status.Errorf(codes.InvalidArgument, "unable to parse command: %s", err.Error())
	}

	if cmd, err = anycmd.UnmarshalNew(); err != nil {
		return status.Errorf(codes.InvalidArgument, "could not unmarshal google.protobuf.Any: %s", err.Error())
	}

	switch cmd := cmd.(type) {
	case *pb.CommandStatementUpdate:
		recordCount, err := f.srv.DoPutCommandStatementUpdate(stream.Context(), cmd)
		if err != nil {
			return err
		}

		result := pb.DoPutUpdateResult{RecordCount: recordCount}
		out := &flight.PutResult{}
		if out.AppMetadata, err = proto.Marshal(&result); err != nil {
			return status.Errorf(codes.Internal, "failed to marshal PutResult: %s", err.Error())
		}
		return stream.Send(out)
	case *pb.CommandPreparedStatementQuery:
		return f.srv.DoPutPreparedStatementQuery(stream.Context(), cmd, rdr, &putMetadataWriter{stream})
	case *pb.CommandPreparedStatementUpdate:
		recordCount, err := f.srv.DoPutPreparedStatementUpdate(stream.Context(), cmd, rdr)
		if err != nil {
			return err
		}

		result := pb.DoPutUpdateResult{RecordCount: recordCount}
		out := &flight.PutResult{}
		if out.AppMetadata, err = proto.Marshal(&result); err != nil {
			return status.Errorf(codes.Internal, "failed to marshal PutResult: %s", err.Error())
		}
		return stream.Send(out)
	default:
		return status.Error(codes.InvalidArgument, "the defined request is invalid")
	}
}

func (f *flightSqlServer) ListActions(_ *flight.Empty, stream flight.FlightService_ListActionsServer) error {
	actions := []string{CreatePreparedStatementActionType, ClosePreparedStatementActionType}

	for _, a := range actions {
		if err := stream.Send(&flight.ActionType{Type: a}); err != nil {
			return err
		}
	}
	return nil
}

func (f *flightSqlServer) DoAction(cmd *flight.Action, stream flight.FlightService_DoActionServer) error {
	var anycmd anypb.Any

	switch cmd.Type {
	case CreatePreparedStatementActionType:
		if err := proto.Unmarshal(cmd.Body, &anycmd); err != nil {
			return status.Errorf(codes.InvalidArgument, "unable to parse command: %s", err.Error())
		}

		var (
			request pb.ActionCreatePreparedStatementRequest
			result  pb.ActionCreatePreparedStatementResult
			ret     pb.Result
		)
		if err := anycmd.UnmarshalTo(&request); err != nil {
			return status.Errorf(codes.InvalidArgument, "unable to unmarshal google.protobuf.Any: %s", err.Error())
		}

		output, err := f.srv.CreatePreparedStatement(stream.Context(), &request)
		if err != nil {
			return err
		}

		result.PreparedStatementHandle = output.Handle
		if output.DatasetSchema != nil {
			result.DatasetSchema = flight.SerializeSchema(output.DatasetSchema, f.mem)
		}
		if output.ParameterSchema != nil {
			result.ParameterSchema = flight.SerializeSchema(output.ParameterSchema, f.mem)
		}

		if err := anycmd.MarshalFrom(&result); err != nil {
			return status.Errorf(codes.Internal, "unable to marshal final response: %s", err.Error())
		}

		if ret.Body, err = proto.Marshal(&anycmd); err != nil {
			return status.Errorf(codes.Internal, "unable to marshal result: %s", err.Error())
		}
		return stream.Send(&ret)
	case ClosePreparedStatementActionType:
		if err := proto.Unmarshal(cmd.Body, &anycmd); err != nil {
			return status.Errorf(codes.InvalidArgument, "unable to parse command: %s", err.Error())
		}

		var request pb.ActionClosePreparedStatementRequest
		if err := anycmd.UnmarshalTo(&request); err != nil {
			return status.Errorf(codes.InvalidArgument, "unable to unmarshal google.protobuf.Any: %s", err.Error())
		}

		if err := f.srv.ClosePreparedStatement(stream.Context(), &request); err != nil {
			return err
		}

		return stream.Send(&pb.Result{})
	default:
		return status.Error(codes.InvalidArgument, "the defined request is invalid.")
	}
}

var (
	_ Server = (*BaseServer)(nil)
)
