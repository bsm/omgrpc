package omgrpc

import "context"

var contextKeyMethod struct{}

func setContextMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, contextKeyMethod, method)
}

func getContextMethod(ctx context.Context) string {
	return ctx.Value(contextKeyMethod).(string) // should never panic
}
