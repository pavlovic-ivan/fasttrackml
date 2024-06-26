package app

import (
	"context"

	"github.com/G-Research/fasttrackml/pkg/api/aim/api/request"
	"github.com/G-Research/fasttrackml/pkg/api/aim/dao/convertors"
	"github.com/G-Research/fasttrackml/pkg/api/aim/dao/models"
	"github.com/G-Research/fasttrackml/pkg/api/aim/dao/repositories"
	"github.com/G-Research/fasttrackml/pkg/common/api"
)

// Service provides service layer to work with `app` business logic.
type Service struct {
	appRepository repositories.AppRepositoryProvider
}

// NewService creates new Service instance.
func NewService(appRepository repositories.AppRepositoryProvider) *Service {
	return &Service{
		appRepository: appRepository,
	}
}

// Get returns app object.
func (s Service) Get(
	ctx context.Context, namespaceID uint, req *request.GetAppRequest,
) (*models.App, error) {
	app, err := s.appRepository.GetByNamespaceIDAndAppID(ctx, namespaceID, req.ID.String())
	if err != nil {
		return nil, api.NewInternalError("unable to find app by id %q: %s", req.ID, err)
	}
	if app == nil {
		return nil, api.NewResourceDoesNotExistError("app '%s' not found", req.ID)
	}
	return app, nil
}

// Create creates new app object.
func (s Service) Create(
	ctx context.Context, namespaceID uint, req *request.CreateAppRequest,
) (*models.App, error) {
	app := convertors.ConvertCreateAppRequestToDBModel(namespaceID, req)
	if err := s.appRepository.Create(ctx, app); err != nil {
		return nil, api.NewInternalError("unable to create app: %v", err)
	}
	return app, nil
}

// Update updates existing app object.
func (s Service) Update(
	ctx context.Context, namespaceID uint, req *request.UpdateAppRequest,
) (*models.App, error) {
	app, err := s.appRepository.GetByNamespaceIDAndAppID(ctx, namespaceID, req.ID.String())
	if err != nil {
		return nil, api.NewInternalError("unable to find app by id %s: %s", req.ID, err)
	}
	if app == nil {
		return nil, api.NewResourceDoesNotExistError("app with id '%s' not found", req.ID)
	}

	app.Type = req.Type
	app.State = models.AppState(req.State)

	if err := s.appRepository.Update(ctx, app); err != nil {
		return nil, api.NewInternalError("unable to update app '%s': %s", app.ID, err)
	}
	return app, nil
}

// GetApps returns the list of active apps.
func (s Service) GetApps(ctx context.Context, namespaceID uint) ([]models.App, error) {
	apps, err := s.appRepository.GetActiveAppsByNamespace(ctx, namespaceID)
	if err != nil {
		return nil, api.NewInternalError("unable to get active apps: %v", err)
	}
	return apps, nil
}

// Delete deletes existing object.
func (s Service) Delete(ctx context.Context, namespaceID uint, req *request.DeleteAppRequest) error {
	app, err := s.appRepository.GetByNamespaceIDAndAppID(ctx, namespaceID, req.ID.String())
	if err != nil {
		return api.NewInternalError("unable to find app by id %s: %s", req.ID, err)
	}
	if app == nil {
		return api.NewResourceDoesNotExistError("app with id '%s' not found", req.ID)
	}

	if err := s.appRepository.Delete(ctx, app); err != nil {
		return api.NewInternalError("unable to delete app by id %s: %s", req.ID, err)
	}
	return nil
}
