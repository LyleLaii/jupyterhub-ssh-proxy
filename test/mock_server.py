from functools import wraps
from flask import abort, jsonify, Flask, request, Response

app = Flask(__name__)
app.config['JSON_AS_ASCII'] = False

def check_auth(auth):
    """This function is called to check if a username /
    password combination is valid.
    """
    return auth == 'token test'
def authenticate():
    """Sends a 403 response that enables basic auth"""
    return Response(
    'Could not verify your access level for that URL.\n'
    'You have to login with proper credentials', 403,
    {'WWW-Authenticate': 'Basic realm="Login Required"'})

def requires_auth(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        auth = request.headers.get('Authorization', None)
        if not auth or not check_auth(auth):
            return authenticate()
        return f(*args, **kwargs)
    return decorated


@app.route('/hub/api/users/<user_name>', methods=['GET'])
@requires_auth
def user_info(user_name):
    if user_name not in ['test', 'test1']:
        return Response('Client Not Found.\n', 404)

    info = {'kind': 'user',
            'name': user_name,
            'admin': True,
            'groups': [],
            'server': '/user/{}/'.format(user_name),
            'pending': None,
            'created': '2022-01-13T12:46:10.251047Z',
            'last_activity': '2022-07-01T09:13:42.130971Z',
            'servers': {'': {'name': '',
                            'last_activity': '2022-07-01T09:13:05.146000Z',
                            'started': '2022-06-26T13:20:45.104152Z',
                            'pending': None,
                            'ready': True,
                            'state': {'pod_name': 'jupyter-{}'.format(user_name)},
                            'url': '/user/{}/'.format(user_name),
                            'user_options': {'profile': 'ml-env'},
                            'progress_url': '/hub/api/users/{}/server/progress'.format(user_name)}
            },
            'auth_state': None}

    if user_name == 'test1':
        info.update({'servers':{}})


    return jsonify(info)


@app.route('/hub/api/proxy', methods=['GET'])
@requires_auth
def proxy():
    info = {"/": {"routespec": "/",
                "target": "http://hub:8081",
                "data": {"hub": True, 
                        "last_activity": "2022-07-03T09:50:24.613Z"}
                },
            "/user/test/": {"routespec": "/user/test/",
                "target": "http://10.0.12.30:8888",
                "data": {"user": "test",
                        "server_name": "",
                        "last_activity": "2022-07-03T09:53:55.092Z"}
                }
            }


    return jsonify(info)

if __name__ == '__main__':
    app.run(host = '0.0.0.0',port = 6868,debug = True)