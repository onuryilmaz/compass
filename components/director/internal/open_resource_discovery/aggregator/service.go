package aggregator

import (
	"context"
	"github.com/kyma-incubator/compass/components/director/pkg/apperrors"
	"strings"

	"github.com/kyma-incubator/compass/components/director/internal/domain/tenant"
	"github.com/kyma-incubator/compass/components/director/internal/model"
	"github.com/kyma-incubator/compass/components/director/internal/open_resource_discovery"
	"github.com/kyma-incubator/compass/components/director/pkg/persistence"
	"github.com/pkg/errors"
)

type service struct {
	transact persistence.Transactioner

	appSvc       ApplicationService
	webhookSvc   WebhookService
	bundleSvc    BundleService
	packageSvc   PackageService
	productSvc   ProductService
	vendorSvc    VendorService
	tombstoneSvc TombstoneService

	ordClient *client
}

func NewService(transact persistence.Transactioner, appSvc ApplicationService, webhookSvc WebhookService, bundleSvc BundleService, packageSvc PackageService, productSvc ProductService, vendorSvc VendorService, tombstoneSvc TombstoneService, client *client) *service {
	return &service{
		transact:     transact,
		appSvc:       appSvc,
		webhookSvc:   webhookSvc,
		bundleSvc:    bundleSvc,
		packageSvc:   packageSvc,
		productSvc:   productSvc,
		vendorSvc:    vendorSvc,
		tombstoneSvc: tombstoneSvc,
		ordClient:    client,
	}
}

func (s *service) SyncORDDocuments(ctx context.Context) error {
	tx, err := s.transact.Begin()
	if err != nil {
		return err
	}
	defer s.transact.RollbackUnlessCommitted(ctx, tx)
	ctx = persistence.SaveToContext(ctx, tx)

	pageCount := 1
	pageSize := 200
	page, err := s.appSvc.ListGlobal(ctx, pageSize, "")
	if err != nil {
		return errors.Wrapf(err, "error while fetching application page number %d", pageCount)
	}
	pageCount++
	if err := s.processAppPage(ctx, page.Data); err != nil {
		return errors.Wrapf(err, "error while processing application page number %d", pageCount)
	}

	for page.PageInfo.HasNextPage {
		page, err = s.appSvc.ListGlobal(ctx, pageSize, page.PageInfo.EndCursor)
		if err != nil {
			return errors.Wrapf(err, "error while fetching page number %d", pageCount)
		}
		if err := s.processAppPage(ctx, page.Data); err != nil {
			return errors.Wrapf(err, "error while processing page number %d", pageCount)
		}
		pageCount++
	}

	return tx.Commit()
}

func (s *service) processAppPage(ctx context.Context, page []*model.Application) error {
	for _, app := range page {
		ctx = tenant.SaveToContext(ctx, app.Tenant, "")
		webhooks, err := s.webhookSvc.List(ctx, app.ID)
		if err != nil {
			return errors.Wrapf(err, "error fetching webhooks for app with id %q", app.ID)
		}
		documents := make([]*open_resource_discovery.Document, 0, 0)
		for _, wh := range webhooks {
			if wh.Type == model.WebhookTypeOpenResourceDiscovery {
				docs, err := s.ordClient.FetchOpenDiscoveryDocuments(ctx, wh.URL)
				if err != nil {
					return errors.Wrapf(err, "error fetching ORD document for webhook with id %q for app with id %q", wh.ID, app.ID)
				}
				documents = append(documents, docs...)
			}
		}
		if err := s.processDocuments(ctx, app.ID, documents); err != nil {
			return errors.Wrapf(err, "error processing ORD documents for app with id %q", app.ID)
		}
	}
	return nil
}

func (s *service) processDocuments(ctx context.Context, appID string, documents open_resource_discovery.Documents) error {
	if err := documents.Validate(); err != nil {
		return errors.Wrap(err, "invalid documents")
	}

	packagesFromDB, err := s.packageSvc.ListByApplicationID(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "error while listing packages for app with id %q", appID)
	}

	bundlesFromDB, err := s.bundleSvc.ListByApplicationIDNoPaging(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "error while listing bundles for app with id %q", appID)
	}

	productsFromDB, err := s.productSvc.ListByApplicationID(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "error while listing products for app with id %q", appID)
	}

	vendorsFromDB, err := s.vendorSvc.ListByApplicationID(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "error while listing vendors for app with id %q", appID)
	}

	tombstonesFromDB, err := s.tombstoneSvc.ListByApplicationID(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "error while listing tombstones for app with id %q", appID)
	}

	for _, doc := range documents {
		for _, pkg := range doc.Packages {
			i, ok := searchInSlice(len(packagesFromDB), func(i int) bool {
				return packagesFromDB[i].OrdID == pkg.OrdID
			})

			if !ok {
				if _, err := s.packageSvc.Create(ctx, appID, pkg); err != nil {
					return err
				}
			} else {
				if err := s.packageSvc.Update(ctx, packagesFromDB[i].ID, pkg); err != nil {
					return err
				}
			}
		}

		for _, bndl := range doc.ConsumptionBundles {
			i, ok := searchInSlice(len(bundlesFromDB), func(i int) bool {
				return bundlesFromDB[i].OrdID == bndl.OrdID
			})

			if !ok {
				if _, err := s.bundleSvc.Create(ctx, appID, bndl); err != nil {
					return err
				}
			} else {
				if err := s.bundleSvc.Update(ctx, bundlesFromDB[i].ID, bundleUpdateInputFromCreateInput(bndl)); err != nil {
					return err
				}
			}
		}

		for _, product := range doc.Products {
			i, ok := searchInSlice(len(productsFromDB), func(i int) bool {
				return productsFromDB[i].OrdID == product.OrdID
			})

			if !ok {
				if _, err := s.productSvc.Create(ctx, appID, product); err != nil {
					return err
				}
			} else {
				if err := s.productSvc.Update(ctx, productsFromDB[i].OrdID, product); err != nil {
					return err
				}
			}
		}

		for _, vendor := range doc.Vendors {
			i, ok := searchInSlice(len(vendorsFromDB), func(i int) bool {
				return vendorsFromDB[i].OrdID == vendor.OrdID
			})

			if !ok {
				if _, err := s.vendorSvc.Create(ctx, appID, vendor); err != nil {
					return err
				}
			} else {
				if err := s.vendorSvc.Update(ctx, vendorsFromDB[i].OrdID, vendor); err != nil {
					return err
				}
			}
		}

		for _, tombstone := range doc.Tombstones {
			i, ok := searchInSlice(len(tombstonesFromDB), func(i int) bool {
				return tombstonesFromDB[i].OrdID == tombstone.OrdID
			})

			if !ok {
				if _, err := s.tombstoneSvc.Create(ctx, appID, tombstone); err != nil {
					return err
				}

				resourceType := strings.Split(tombstone.OrdID, ":")[1]
				switch resourceType {
				case "package":
					if i, ok := searchInSlice(len(packagesFromDB), func(i int) bool {
						return packagesFromDB[i].OrdID == tombstone.OrdID
					}); ok {
						if err = s.packageSvc.Delete(ctx, packagesFromDB[i].ID); err != nil {
							return err
						}
					}
				case "apiResource":
				case "eventResource":
				case "vendor":
					if err = s.vendorSvc.Delete(ctx, tombstone.OrdID); err != nil && !apperrors.IsNotFoundError(err) {
						return err
					}
				case "product":
					if err = s.productSvc.Delete(ctx, tombstone.OrdID); err != nil && !apperrors.IsNotFoundError(err) {
						return err
					}
				case "consumptionBundle":
					if i, ok := searchInSlice(len(bundlesFromDB), func(i int) bool {
						return *bundlesFromDB[i].OrdID == tombstone.OrdID
					}); ok {
						if err = s.bundleSvc.Delete(ctx, bundlesFromDB[i].ID); err != nil {
							return err
						}
					}
				}
			} else {
				if err := s.tombstoneSvc.Update(ctx, tombstonesFromDB[i].OrdID, tombstone); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func bundleUpdateInputFromCreateInput(in model.BundleCreateInput) model.BundleUpdateInput {
	return model.BundleUpdateInput{
		Name:                           in.Name,
		Description:                    in.Description,
		InstanceAuthRequestInputSchema: in.InstanceAuthRequestInputSchema,
		DefaultInstanceAuth:            in.DefaultInstanceAuth,
		OrdID:                          in.OrdID,
		ShortDescription:               in.ShortDescription,
		Links:                          in.Links,
		Labels:                         in.Labels,
		CredentialExchangeStrategies:   in.CredentialExchangeStrategies,
	}
}

func searchInSlice(length int, f func(i int) bool) (int, bool) {
	for i := 0; i < length; i++ {
		if f(i) {
			return i, true
		}
	}
	return -1, false
}
