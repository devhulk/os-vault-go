auto_auth {
  method "approle" {
    config = {
      role_id_file_path   = "role_id_payments"
      secret_id_file_path = "secret_id_payments"
      remove_secret_id_file_after_reading = false
    }
  }

  sink "file" {
    config = {
      path = "../vault/secrets/token"
    }
  }

}
