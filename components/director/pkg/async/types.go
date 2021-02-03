package async

import (
	"context"

	"github.com/kyma-incubator/compass/components/op-controller/api/v1beta1"
)

type Scheduler interface {
	Schedule(ctx context.Context, op *v1beta1.Operation) error
	Watch(ctx context.Context) error
}