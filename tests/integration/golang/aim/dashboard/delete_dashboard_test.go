package run

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/G-Research/fasttrackml/pkg/common/api"
	"github.com/G-Research/fasttrackml/pkg/database"
	"github.com/G-Research/fasttrackml/tests/integration/golang/helpers"
)

type DeleteDashboardTestSuite struct {
	helpers.BaseTestSuite
}

func TestDeleteDashboardTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteDashboardTestSuite))
}

func (s *DeleteDashboardTestSuite) Test_Ok() {
	dashboard, err := s.DashboardFixtures.CreateDashboard(context.Background(), &database.Dashboard{
		Name: "dashboard-exp",
		App: database.App{
			Type:        "mpi",
			State:       database.AppState{},
			NamespaceID: s.DefaultNamespace.ID,
		},
		Description: "dashboard for experiment",
	})
	s.Require().Nil(err)

	tests := []struct {
		name                   string
		expectedDashboardCount int
	}{
		{
			name:                   "DeleteDashboard",
			expectedDashboardCount: 0,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var resp api.ErrorResponse
			s.Require().Nil(
				s.AIMClient().WithMethod(
					http.MethodDelete,
				).WithResponse(
					&resp,
				).DoRequest(
					"/dashboards/%s", dashboard.ID,
				),
			)
			dashboards, err := s.DashboardFixtures.GetDashboards(context.Background())
			s.Require().Nil(err)
			s.Equal(tt.expectedDashboardCount, len(dashboards))
		})
	}
}

func (s *DeleteDashboardTestSuite) Test_Error() {
	_, err := s.DashboardFixtures.CreateDashboard(context.Background(), &database.Dashboard{
		Name: "dashboard-exp",
		App: database.App{
			Type:        "mpi",
			State:       database.AppState{},
			NamespaceID: s.DefaultNamespace.ID,
		},
		Description: "dashboard for experiment",
	})
	s.Require().Nil(err)

	tests := []struct {
		name                   string
		idParam                uuid.UUID
		expectedDashboardCount int
	}{
		{
			name:                   "DeleteDashboardWithNotFoundID",
			idParam:                uuid.New(),
			expectedDashboardCount: 1,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var resp api.ErrorResponse
			s.Require().Nil(
				s.AIMClient().WithMethod(
					http.MethodDelete,
				).WithResponse(
					&resp,
				).DoRequest(
					"/dashboards/%s", tt.idParam,
				),
			)
			s.Contains(resp.Message, "Not Found")

			dashboards, err := s.DashboardFixtures.GetDashboards(context.Background())
			s.Require().Nil(err)
			s.Equal(tt.expectedDashboardCount, len(dashboards))
		})
	}
}
