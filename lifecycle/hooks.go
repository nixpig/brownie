package lifecycle

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func ExecHooks(hooks []specs.Hook, state string, log *zerolog.Logger) error {
	for _, h := range hooks {
		ctx := context.Background()
		if h.Timeout != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(
				ctx,
				time.Duration(*h.Timeout)*time.Second,
			)
			defer cancel()
		}

		args := h.Args[1:]
		args = append(args, state)
		cmd := exec.CommandContext(ctx, h.Path, args...)
		cmd.Env = h.Env
		cmd.Stdin = strings.NewReader(state)

		if out, err := cmd.CombinedOutput(); err != nil {
			log.Error().
				Str("out", string(out)).
				Msg("stderr and stdout")
			return fmt.Errorf("start exec hook: (path: %s) (args: %+v) (out: %s): %w", h.Path, args, string(out), err)
		}

		// if err := cmd.Wait(); err != nil {
		// 	return fmt.Errorf("wait exec hook: %s %+v: %w", h.Path, args, err)
		// }
	}

	return nil
}
