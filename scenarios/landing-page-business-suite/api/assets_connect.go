package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type assetsConnectHandler struct{ service *AssetsService }

func assetProto(asset *Asset) *lpbsv1.Asset {
	if asset == nil {
		return &lpbsv1.Asset{}
	}
	return &lpbsv1.Asset{Id: int64(asset.ID), Filename: asset.Filename, OriginalFilename: asset.OriginalFilename, MimeType: asset.MimeType, SizeBytes: asset.SizeBytes, StoragePath: asset.StoragePath, ThumbnailPath: asset.ThumbnailPath, AltText: asset.AltText, Category: asset.Category, UploadedBy: asset.UploadedBy, CreatedAt: timestamppb.New(asset.CreatedAt), Url: asset.URL, Derivatives: asset.Derivatives}
}

func (h assetsConnectHandler) ListAssets(_ context.Context, request *connect.Request[lpbsv1.ListAssetsRequest]) (*connect.Response[lpbsv1.ListAssetsResponse], error) {
	assets, err := h.service.List(request.Msg.GetCategory())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list assets: %w", err))
	}
	result := &lpbsv1.ListAssetsResponse{}
	for index := range assets {
		result.Assets = append(result.Assets, assetProto(&assets[index]))
	}
	return connect.NewResponse(result), nil
}

func (h assetsConnectHandler) GetAsset(_ context.Context, request *connect.Request[lpbsv1.GetAssetRequest]) (*connect.Response[lpbsv1.AssetResponse], error) {
	if request.Msg.GetId() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id must be positive"))
	}
	asset, err := h.service.Get(int(request.Msg.GetId()))
	if errors.Is(err, ErrAssetNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get asset: %w", err))
	}
	return connect.NewResponse(&lpbsv1.AssetResponse{Asset: assetProto(asset)}), nil
}

func (h assetsConnectHandler) DeleteAsset(ctx context.Context, request *connect.Request[lpbsv1.DeleteAssetRequest]) (*connect.Response[lpbsv1.DeleteAssetResponse], error) {
	if request.Msg.GetId() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id must be positive"))
	}
	if err := h.service.DeleteContext(ctx, int(request.Msg.GetId())); err != nil {
		if errors.Is(err, ErrAssetNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete asset: %w", err))
	}
	return connect.NewResponse(&lpbsv1.DeleteAssetResponse{Deleted: true}), nil
}

func registerAssetsConnectRoutes(router *mux.Router, service *AssetsService, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewAssetsServiceHandler(assetsConnectHandler{service: service})
	for _, procedure := range []string{lpbsconnect.AssetsServiceListAssetsProcedure, lpbsconnect.AssetsServiceGetAssetProcedure, lpbsconnect.AssetsServiceDeleteAssetProcedure} {
		router.Handle(procedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
	}
}

var _ lpbsconnect.AssetsServiceHandler = assetsConnectHandler{}
