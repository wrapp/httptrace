# encoding: utf-8

task :glide do
  sh 'glide install'
end

desc 'Run the unit tests locally'
task :test do
  sh 'go test -v -race -cover'
end

desc 'Generate coverage data for the tests and display it in the default browser'
task :test_coverage do
  begin
    sh 'mkdir coverage'
    sh 'go test -race -coverprofile=coverage/c.out'
    sh 'go tool cover -html=coverage/c.out'
  ensure
    sh 'rm -rf coverage'
  end
end

task :default => [:test]