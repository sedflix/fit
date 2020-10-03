# Service: numbers
docker_build('random_number/numbers', 'numbers',
    live_update=[
        sync('./numbers', '/app'),
        run('cd /app && pip install -r requirements.txt',
            trigger='numbers/requirements.txt'),
    ]
)

#