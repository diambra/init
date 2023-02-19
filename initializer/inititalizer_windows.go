package initializer

import (
	"github.com/go-kit/log"
)

func (i *Initializer) Init(logger log.Logger) error {
	return i.init(logger)
}
