# pathway - A path-based API server written in Go

See article at [http://hire.jonasgalvez.com.br/2018/May/20/You-Dont-Need-REST](http://hire.jonasgalvez.com.br/2018/May/20/You-Dont-Need-REST).

## Build and run

    go build
    ./pathway

## Testing method calls

    curl -H "Content-Type: application/json" \
      -d '{"test": 1}' http://0.0.0.0:4000/api/echo/message

## Adding services

An example EchoService is provided (echo_service.go). A real service may talk to other APIs or 
database services directly. Here's an auth proxy would look like:

    type LoginCredentials struct {
      email    *string `json:"email"`
      password *string `json:"passsword"`
    }

    func (s *AccountsService) Login(
      ctx context.Context,
      payload json.RawMessage,
    ) (
      *json.RawMessage, 
      *Response, 
      error,
    ) {
      log.Println("Accounts.Login called")
      req, err := s.client.NewRequest("POST", "sessions/", &payload)
      if err != nil {
        return nil, nil, err
      }
      data := new(json.RawMessage)
      resp, err := s.client.Do(ctx, req, &data)
      if err != nil {
        return nil, resp, err
      }
      return data, resp, nil
    }

## Meta

Stored E-commerce

Distributed under the MIT license. See ``LICENSE`` for more information.

[https://github.com/stored](https://github.com/stored)

## Contributing

We're very much looking forward to Pull Requests enhancing features or addressing potential 
issues. As we further incorporate this into our infrastructure, we'll update this repo as well.

1. Fork it (<https://github.com/yourname/yourproject/fork>)
2. Create your feature branch (`git checkout -b feature/foo-bar`)
3. Commit your changes (`git commit -am 'Add some foo-bar'`)
4. Push to the branch (`git push origin feature/foo-bar`)
5. Create a new Pull Request
