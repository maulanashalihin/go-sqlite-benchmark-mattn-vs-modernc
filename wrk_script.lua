-- wrk script to randomize user IDs for realistic load testing
math.randomseed(os.time())

request = function()
   local id = math.random(1, 10000)
   return wrk.format(nil, "/users/" .. id)
end
