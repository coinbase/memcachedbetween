require 'dalli'
require 'dalli-elasticache'
require 'active_support'
require 'active_support/cache/dalli_store'
require 'connection_pool'

endpoint    = "localhost:11210"
elasticache = Dalli::ElastiCache.new(endpoint)
unix_servers = elasticache.endpoint.config.text.scan(/(\/[\/\w.-]+)\|\|/).flatten
servers = unix_servers.empty? ? elasticache.servers : unix_servers

puts '!!!!!!!!!!!!!!!!!'
puts servers

store = ActiveSupport::Cache::DalliStore.new(servers, pool_size: 10)

Dalli.logger.level = Logger::DEBUG

20.times do |i|
  store.write("key#{i}", "value#{i}", expires_in: 10)
  puts store.fetch("key#{i}")
end
