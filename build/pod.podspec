Pod::Spec.new do |spec|
  spec.name         = 'Gvap'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/vaporyco/go-vapory'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Vapory Client'
  spec.source       = { :git => 'https://github.com/vaporyco/go-vapory.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gvap.framework'

	spec.prepare_command = <<-CMD
    curl https://gvapstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gvap.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
