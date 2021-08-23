vcl 4.0;

backend default {
  .host = "host.docker.internal:3000";
}

sub vcl_recv {
    if (req.url ~ "^/api/isu") {
        unset req.http.Cookie;
    }
}

sub vcl_backend_response {
    if (bereq.url ~ "^/api/isu") {
      set beresp.ttl = 3s;
    }
}