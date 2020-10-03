load('ext://restart_process', 'docker_build_with_restart')

compile_cmd = 'cd src; go mod download; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fit .'
local_resource( 'compile-main',
    compile_cmd,
    deps=['./src'],
)

docker_build_with_restart(
    ref="sedflix/fit",
    context="./",
    entrypoint="./fit",
    dockerfile="./Dockerfile.tilt",
    live_update=[
        sync('./src/fit', '/root/fit'),
        sync('./src/web', '/root/web'),
        run('cd /root && ./fit', trigger=["./src/fit", './src/web'])
     ]
)


# TODO: change this to whatever context you want to use
allow_k8s_contexts('zone-aws-default-staging-mumbai')

yaml = helm(
  './charts',
  name='fit-tilt',
  namespace='test',
  values=['charts/values.yaml'],
)
watch_file('./charts')
k8s_yaml(yaml)
k8s_resource(workload="fit-tilt", port_forwards=9080, resource_deps=["compile-main"])

