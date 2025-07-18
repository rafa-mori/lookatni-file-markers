# 🔄 CI/CD Integration Examples

Learn how to integrate LookAtni File Markers into your CI/CD pipelines for automated project documentation, testing, and deployment workflows.

## 🎯 Overview

LookAtni File Markers can enhance CI/CD pipelines by providing automated project snapshots, documentation generation, and testing artifacts. This guide covers various integration patterns and use cases.

## 🚀 GitHub Actions Integration

### Basic Workflow with LookAtni

```yaml
# .github/workflows/lookatni-integration.yml
name: LookAtni Integration

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  generate-markers:
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout repository
      uses: actions/checkout@v4
      
    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '18'
        cache: 'npm'
        
    - name: Install dependencies
      run: npm ci
      
    - name: Install LookAtni CLI
      run: npm install -g lookatni-cli
      
    - name: Generate project markers
      run: |
        lookatni generate \
          --source . \
          --output ./artifacts/project-snapshot.lookatni \
          --compress \
          --include-metadata
          
    - name: Upload markers as artifact
      uses: actions/upload-artifact@v4
      with:
        name: project-markers-${{ github.sha }}
        path: ./artifacts/project-snapshot.lookatni
        retention-days: 30
        
    - name: Validate generated markers
      run: |
        lookatni validate ./artifacts/project-snapshot.lookatni
        
    - name: Generate statistics
      run: |
        lookatni stats ./artifacts/project-snapshot.lookatni > ./artifacts/project-stats.json
        
    - name: Comment PR with statistics
      if: github.event_name == 'pull_request'
      uses: actions/github-script@v7
      with:
        script: |
          const fs = require('fs');
          const stats = JSON.parse(fs.readFileSync('./artifacts/project-stats.json', 'utf8'));
          
          const comment = `## 📊 Project Statistics
          
          | Metric | Value |
          |--------|-------|
          | Total Files | ${stats.totalFiles} |
          | Code Files | ${stats.codeFiles} |
          | Total Size | ${stats.totalSize} |
          | Complexity Score | ${stats.complexityScore} |
          
          Generated by LookAtni File Markers 🎯`;
          
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: comment
          });
```

### Advanced Workflow with Matrix Strategy

```yaml
# .github/workflows/multi-env-markers.yml
name: Multi-Environment Markers

on:
  workflow_dispatch:
    inputs:
      environments:
        description: 'Environments to process'
        required: true
        default: 'dev,staging,prod'
        type: string

jobs:
  generate-environment-markers:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: [dev, staging, prod]
        
    steps:
    - uses: actions/checkout@v4
      
    - name: Setup environment-specific configuration
      run: |
        echo "Setting up for environment: ${{ matrix.environment }}"
        cp configs/${{ matrix.environment }}.env .env
        
    - name: Generate environment markers
      run: |
        lookatni generate \
          --source . \
          --output ./markers/${{ matrix.environment }}-snapshot.lookatni \
          --config ./configs/${{ matrix.environment }}.lookatni.json \
          --tag "env:${{ matrix.environment }}" \
          --tag "build:${{ github.run_number }}"
          
    - name: Upload environment markers
      uses: actions/upload-artifact@v4
      with:
        name: markers-${{ matrix.environment }}
        path: ./markers/${{ matrix.environment }}-snapshot.lookatni
        
  create-release-bundle:
    needs: generate-environment-markers
    runs-on: ubuntu-latest
    
    steps:
    - name: Download all artifacts
      uses: actions/download-artifact@v4
      
    - name: Create release bundle
      run: |
        mkdir -p release-bundle
        
        # Combine all environment markers
        lookatni combine \
          --input "markers-*/**.lookatni" \
          --output release-bundle/complete-release-${{ github.run_number }}.lookatni \
          --metadata "release_number:${{ github.run_number }}" \
          --metadata "release_date:$(date -u +%Y-%m-%dT%H:%M:%SZ)"
          
    - name: Upload release bundle
      uses: actions/upload-artifact@v4
      with:
        name: release-bundle-${{ github.run_number }}
        path: release-bundle/
        retention-days: 90
```

## 🔧 GitLab CI Integration

### GitLab Pipeline Configuration

```yaml
# .gitlab-ci.yml
stages:
  - validate
  - build
  - test
  - documentation
  - deploy

variables:
  LOOKATNI_VERSION: "latest"
  NODE_VERSION: "18"

.lookatni-base:
  image: node:${NODE_VERSION}
  before_script:
    - npm install -g lookatni-cli@${LOOKATNI_VERSION}
  cache:
    paths:
      - node_modules/

validate-project:
  extends: .lookatni-base
  stage: validate
  script:
    - lookatni validate-project --source . --strict
    - lookatni check-dependencies --report-format json > dependency-report.json
  artifacts:
    reports:
      junit: dependency-report.json
    paths:
      - dependency-report.json
    expire_in: 1 week

generate-markers:
  extends: .lookatni-base
  stage: build
  script:
    - |
      lookatni generate \
        --source . \
        --output artifacts/project-$CI_COMMIT_SHA.lookatni \
        --compress \
        --include-git-info \
        --metadata "pipeline_id:$CI_PIPELINE_ID" \
        --metadata "commit_sha:$CI_COMMIT_SHA" \
        --metadata "branch:$CI_COMMIT_REF_NAME"
  artifacts:
    paths:
      - artifacts/project-$CI_COMMIT_SHA.lookatni
    expire_in: 1 month

test-with-markers:
  extends: .lookatni-base
  stage: test
  dependencies:
    - generate-markers
  script:
    - |
      # Create isolated test environment from markers
      mkdir test-env
      lookatni extract \
        --input artifacts/project-$CI_COMMIT_SHA.lookatni \
        --output test-env/ \
        --validate-integrity
        
    - cd test-env
    - npm ci
    - npm test
  coverage: '/Coverage: \d+\.\d+%/'

documentation:
  extends: .lookatni-base
  stage: documentation
  script:
    - |
      # Generate documentation with project structure
      lookatni doc-generate \
        --source . \
        --output docs/auto-generated/ \
        --format markdown \
        --include-tree \
        --include-stats
        
    - |
      # Create interactive project explorer
      lookatni web-viewer \
        --input artifacts/project-$CI_COMMIT_SHA.lookatni \
        --output public/project-explorer/ \
        --theme dark
  artifacts:
    paths:
      - docs/auto-generated/
      - public/project-explorer/
    expire_in: 1 week
  only:
    - main
    - develop

deploy-markers:
  extends: .lookatni-base
  stage: deploy
  script:
    - |
      # Deploy to artifact repository
      lookatni upload \
        --input artifacts/project-$CI_COMMIT_SHA.lookatni \
        --repository $ARTIFACT_REPOSITORY_URL \
        --token $ARTIFACT_REPOSITORY_TOKEN \
        --tag "version:$CI_COMMIT_TAG" \
        --tag "environment:production"
  only:
    - tags
```

### Branch-Specific Workflows

```yaml
# .gitlab-ci.yml (continued)
feature-branch-analysis:
  extends: .lookatni-base
  stage: validate
  script:
    - |
      # Compare with main branch
      git fetch origin main
      
      lookatni diff \
        --base origin/main \
        --head HEAD \
        --output feature-analysis.json \
        --format json
        
    - |
      # Generate change summary
      lookatni summarize-changes \
        --input feature-analysis.json \
        --output change-summary.md \
        --format markdown
        
    - |
      # Post comment to merge request
      lookatni gitlab-comment \
        --merge-request $CI_MERGE_REQUEST_IID \
        --comment-file change-summary.md \
        --project-id $CI_PROJECT_ID \
        --token $GITLAB_TOKEN
  only:
    - merge_requests
  artifacts:
    paths:
      - feature-analysis.json
      - change-summary.md
    expire_in: 1 week
```

## 🏗️ Jenkins Pipeline Integration

### Declarative Pipeline

```groovy
// Jenkinsfile
pipeline {
    agent any
    
    environment {
        LOOKATNI_CLI = 'lookatni'
        NODE_VERSION = '18'
    }
    
    tools {
        nodejs "${NODE_VERSION}"
    }
    
    stages {
        stage('Setup') {
            steps {
                script {
                    // Install LookAtni CLI
                    sh 'npm install -g lookatni-cli'
                    
                    // Verify installation
                    sh 'lookatni --version'
                }
            }
        }
        
        stage('Pre-build Analysis') {
            parallel {
                stage('Generate Baseline') {
                    steps {
                        script {
                            sh """
                                ${LOOKATNI_CLI} generate \\
                                    --source . \\
                                    --output baseline-${BUILD_NUMBER}.lookatni \\
                                    --exclude-patterns 'node_modules/**,dist/**,*.log' \\
                                    --metadata 'build_number:${BUILD_NUMBER}' \\
                                    --metadata 'jenkins_job:${JOB_NAME}'
                            """
                        }
                    }
                }
                
                stage('Validate Project Structure') {
                    steps {
                        script {
                            sh """
                                ${LOOKATNI_CLI} validate-structure \\
                                    --source . \\
                                    --rules .lookatni-rules.json \\
                                    --report-format junit \\
                                    --output validation-report.xml
                            """
                        }
                    }
                    post {
                        always {
                            publishTestResults testResultsPattern: 'validation-report.xml'
                        }
                    }
                }
            }
        }
        
        stage('Build') {
            steps {
                sh 'npm ci'
                sh 'npm run build'
            }
        }
        
        stage('Test') {
            steps {
                script {
                    // Create test environment from markers
                    sh """
                        mkdir -p test-environments/clean
                        ${LOOKATNI_CLI} extract \\
                            --input baseline-${BUILD_NUMBER}.lookatni \\
                            --output test-environments/clean/ \\
                            --verify-checksums
                    """
                    
                    // Run tests in clean environment
                    dir('test-environments/clean') {
                        sh 'npm test'
                    }
                }
            }
        }
        
        stage('Generate Artifacts') {
            when {
                anyOf {
                    branch 'main'
                    branch 'develop'
                    buildingTag()
                }
            }
            steps {
                script {
                    // Generate release markers
                    sh """
                        ${LOOKATNI_CLI} generate \\
                            --source . \\
                            --output release-${BUILD_NUMBER}.lookatni \\
                            --compress \\
                            --include-build-info \\
                            --metadata 'release_candidate:true' \\
                            --metadata 'tested:true'
                    """
                    
                    // Generate documentation
                    sh """
                        ${LOOKATNI_CLI} doc-extract \\
                            --input release-${BUILD_NUMBER}.lookatni \\
                            --output docs/api/ \\
                            --format html \\
                            --include-examples
                    """
                }
            }
        }
        
        stage('Deploy to Staging') {
            when {
                branch 'develop'
            }
            steps {
                script {
                    // Deploy using markers
                    sh """
                        ${LOOKATNI_CLI} deploy \\
                            --input release-${BUILD_NUMBER}.lookatni \\
                            --target staging \\
                            --config deploy-staging.json \\
                            --verify-deployment
                    """
                }
            }
        }
        
        stage('Deploy to Production') {
            when {
                buildingTag()
            }
            steps {
                script {
                    // Production deployment with approval
                    input message: 'Deploy to production?', 
                          parameters: [
                              choice(name: 'DEPLOY_STRATEGY', 
                                    choices: ['blue-green', 'rolling', 'canary'],
                                    description: 'Deployment strategy')
                          ]
                    
                    sh """
                        ${LOOKATNI_CLI} deploy \\
                            --input release-${BUILD_NUMBER}.lookatni \\
                            --target production \\
                            --strategy ${DEPLOY_STRATEGY} \\
                            --config deploy-production.json \\
                            --create-rollback-point
                    """
                }
            }
        }
    }
    
    post {
        always {
            // Archive artifacts
            archiveArtifacts artifacts: '*.lookatni, docs/**', 
                            allowEmptyArchive: true
            
            // Cleanup
            sh 'rm -rf test-environments/'
        }
        
        success {
            script {
                // Notification with project stats
                def stats = sh(
                    script: "${LOOKATNI_CLI} stats release-${BUILD_NUMBER}.lookatni --format json",
                    returnStdout: true
                ).trim()
                
                slackSend(
                    color: 'good',
                    message: """
                        ✅ Build ${BUILD_NUMBER} successful!
                        
                        📊 Project Stats:
                        ```
                        ${stats}
                        ```
                    """
                )
            }
        }
        
        failure {
            slackSend(
                color: 'danger',
                message: "❌ Build ${BUILD_NUMBER} failed! Check Jenkins for details."
            )
        }
    }
}
```

### Pipeline Library Integration

```groovy
// vars/lookatniPipeline.groovy
def call(Map config) {
    pipeline {
        agent any
        
        environment {
            LOOKATNI_VERSION = config.lookatniVersion ?: 'latest'
        }
        
        stages {
            stage('LookAtni Setup') {
                steps {
                    script {
                        lookatniSetup(config)
                    }
                }
            }
            
            stage('Generate Markers') {
                steps {
                    script {
                        generateProjectMarkers(config)
                    }
                }
            }
            
            stage('Quality Gates') {
                parallel {
                    stage('Structure Validation') {
                        steps {
                            script {
                                validateProjectStructure(config)
                            }
                        }
                    }
                    
                    stage('Dependency Analysis') {
                        steps {
                            script {
                                analyzeDependencies(config)
                            }
                        }
                    }
                    
                    stage('Security Scan') {
                        steps {
                            script {
                                securityScanWithLookatni(config)
                            }
                        }
                    }
                }
            }
            
            stage('Build & Test') {
                steps {
                    script {
                        buildAndTestWithMarkers(config)
                    }
                }
            }
            
            stage('Release Preparation') {
                when {
                    anyOf {
                        branch 'main'
                        buildingTag()
                    }
                }
                steps {
                    script {
                        prepareRelease(config)
                    }
                }
            }
        }
        
        post {
            always {
                script {
                    lookatniCleanup(config)
                }
            }
            
            success {
                script {
                    lookatniNotification(config, 'success')
                }
            }
            
            failure {
                script {
                    lookatniNotification(config, 'failure')
                }
            }
        }
    }
}

def generateProjectMarkers(config) {
    sh """
        lookatni generate \\
            --source ${config.sourceDir ?: '.'} \\
            --output markers/project-${BUILD_NUMBER}.lookatni \\
            --config ${config.lookatniConfig ?: '.lookatni.json'} \\
            --metadata 'build_number:${BUILD_NUMBER}' \\
            --metadata 'branch:${BRANCH_NAME}' \\
            --metadata 'commit:${GIT_COMMIT}'
    """
}
```

## 🐳 Docker Integration

### Multi-Stage Docker Build with LookAtni

```dockerfile
# Dockerfile
FROM node:18-alpine AS base
WORKDIR /app

# Install LookAtni CLI
RUN npm install -g lookatni-cli

# Copy source code
COPY . .

# Generate project markers for build reproducibility
RUN lookatni generate \
    --source . \
    --output /tmp/build-source.lookatni \
    --exclude-patterns 'node_modules/**,dist/**' \
    --metadata "docker_build:true" \
    --metadata "build_timestamp:$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Build stage
FROM base AS builder
RUN npm ci --only=production

# Verify build environment matches source
RUN lookatni extract \
    --input /tmp/build-source.lookatni \
    --output /tmp/verify-source \
    --verify-only

RUN npm run build

# Production stage
FROM node:18-alpine AS production
WORKDIR /app

# Copy built application
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json

# Copy build markers for runtime verification
COPY --from=builder /tmp/build-source.lookatni ./markers/

# Health check using LookAtni
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD lookatni verify-runtime \
        --markers ./markers/build-source.lookatni \
        --current-state /app \
        --critical-files "dist/index.js,package.json"

EXPOSE 3000
CMD ["node", "dist/index.js"]
```

### Docker Compose with LookAtni Services

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "3000:3000"
    volumes:
      - ./markers:/app/markers
    environment:
      - LOOKATNI_VERIFY_ON_START=true
    healthcheck:
      test: ["CMD", "lookatni", "health-check", "/app"]
      interval: 30s
      timeout: 10s
      retries: 3

  marker-monitor:
    image: node:18-alpine
    volumes:
      - .:/workspace
      - markers-data:/markers
    working_dir: /workspace
    command: |
      sh -c "
        npm install -g lookatni-cli &&
        lookatni monitor-changes \
          --source /workspace \
          --output /markers \
          --interval 300 \
          --webhook http://notification-service:8080/markers
      "
    depends_on:
      - app

  documentation:
    image: node:18-alpine
    volumes:
      - .:/workspace
      - ./docs:/docs
    working_dir: /workspace
    command: |
      sh -c "
        npm install -g lookatni-cli &&
        lookatni doc-server \
          --source /workspace \
          --output /docs \
          --port 8080 \
          --auto-reload
      "
    ports:
      - "8080:8080"

volumes:
  markers-data:
```

## ⚡ Azure DevOps Integration

### Azure Pipeline YAML

```yaml
# azure-pipelines.yml
trigger:
  branches:
    include:
      - main
      - develop
      - feature/*

pool:
  vmImage: 'ubuntu-latest'

variables:
  - group: lookatni-config
  - name: nodeVersion
    value: '18.x'

stages:
- stage: ValidateAndGenerate
  displayName: 'Validate and Generate Markers'
  jobs:
  - job: LookatniValidation
    displayName: 'LookAtni Validation'
    steps:
    - task: NodeTool@0
      inputs:
        versionSpec: $(nodeVersion)
      displayName: 'Setup Node.js'

    - script: |
        npm install -g lookatni-cli
        lookatni --version
      displayName: 'Install LookAtni CLI'

    - script: |
        lookatni validate-project \
          --source $(System.DefaultWorkingDirectory) \
          --config .lookatni-azure.json \
          --output validation-report.json
      displayName: 'Validate Project Structure'

    - task: PublishTestResults@2
      inputs:
        testResultsFormat: 'JUnit'
        testResultsFiles: 'validation-report.xml'
      condition: always()

    - script: |
        lookatni generate \
          --source $(System.DefaultWorkingDirectory) \
          --output $(Build.ArtifactStagingDirectory)/project-markers.lookatni \
          --compress \
          --metadata "build_id:$(Build.BuildId)" \
          --metadata "source_branch:$(Build.SourceBranch)" \
          --metadata "agent_name:$(Agent.Name)"
      displayName: 'Generate Project Markers'

    - task: PublishBuildArtifacts@1
      inputs:
        pathToPublish: '$(Build.ArtifactStagingDirectory)'
        artifactName: 'lookatni-markers'
      displayName: 'Publish Markers Artifact'

- stage: BuildAndTest
  displayName: 'Build and Test'
  dependsOn: ValidateAndGenerate
  jobs:
  - job: BuildWithMarkers
    displayName: 'Build with LookAtni Verification'
    steps:
    - task: DownloadBuildArtifacts@0
      inputs:
        artifactName: 'lookatni-markers'
        downloadPath: '$(System.ArtifactsDirectory)'

    - script: |
        # Verify source integrity
        lookatni verify-integrity \
          --markers $(System.ArtifactsDirectory)/lookatni-markers/project-markers.lookatni \
          --source $(System.DefaultWorkingDirectory)
      displayName: 'Verify Source Integrity'

    - script: |
        npm ci
        npm run build
        npm test
      displayName: 'Build and Test'

    - script: |
        # Generate post-build markers
        lookatni generate \
          --source $(System.DefaultWorkingDirectory) \
          --output $(Build.ArtifactStagingDirectory)/post-build-markers.lookatni \
          --include-build-artifacts \
          --diff-from $(System.ArtifactsDirectory)/lookatni-markers/project-markers.lookatni
      displayName: 'Generate Post-Build Markers'

- stage: Release
  displayName: 'Release'
  condition: and(succeeded(), eq(variables['Build.SourceBranch'], 'refs/heads/main'))
  dependsOn: BuildAndTest
  jobs:
  - deployment: DeployWithLookatni
    displayName: 'Deploy with LookAtni Tracking'
    environment: 'production'
    strategy:
      runOnce:
        deploy:
          steps:
          - script: |
              lookatni deploy-prepare \
                --markers $(System.ArtifactsDirectory)/lookatni-markers/post-build-markers.lookatni \
                --environment production \
                --output deployment-package.tar.gz
            displayName: 'Prepare Deployment Package'

          - script: |
              # Deploy application
              echo "Deploying application..."
              
              # Verify deployment
              lookatni verify-deployment \
                --package deployment-package.tar.gz \
                --target production \
                --health-check-url https://myapp.com/health
            displayName: 'Deploy and Verify'
```

## 🔍 Advanced CI/CD Patterns

### Marker-Based Feature Flags

```typescript
// feature-flag-manager.ts
class MarkerBasedFeatureFlags {
    async generateFeatureBranch(
        baseBranch: string,
        featureName: string,
        features: string[]
    ): Promise<void> {
        // Generate markers for base branch
        const baseMarkers = await this.generateMarkers(baseBranch);
        
        // Apply feature flags
        const featureMarkers = await this.applyFeatureFlags(
            baseMarkers,
            features
        );
        
        // Deploy feature branch
        await this.deployWithMarkers(featureMarkers, `feature-${featureName}`);
    }

    private async applyFeatureFlags(
        baseMarkers: string,
        features: string[]
    ): Promise<string> {
        const config = {
            enabledFeatures: features,
            featureFlags: this.getFeatureConfiguration(features)
        };

        return await this.lookatniCLI.transform(
            baseMarkers,
            'apply-feature-flags',
            config
        );
    }
}
```

### Progressive Deployment with Markers

```typescript
// progressive-deployment.ts
class ProgressiveDeployment {
    async executeCanaryDeployment(
        releaseMarkers: string,
        canaryPercentage: number = 10
    ): Promise<void> {
        // Deploy to canary environment
        await this.deployCanary(releaseMarkers, canaryPercentage);
        
        // Monitor metrics
        const metrics = await this.monitorCanary(5); // 5 minutes
        
        if (metrics.errorRate < 0.1 && metrics.responseTime < 200) {
            // Gradually increase traffic
            await this.increaseCanaryTraffic(25);
            await this.monitorCanary(10);
            
            await this.increaseCanaryTraffic(50);
            await this.monitorCanary(10);
            
            // Full deployment
            await this.promoteToProduction(releaseMarkers);
        } else {
            // Rollback
            await this.rollbackCanary();
        }
    }
}
```

---

## 📋 CI/CD Best Practices Summary

### ✅ Integration Checklist

- **Automated marker generation** on every build
- **Validation gates** before deployment
- **Artifact versioning** with meaningful metadata
- **Environment-specific configurations**
- **Rollback capabilities** using markers
- **Security scanning** of generated markers
- **Performance monitoring** of CI/CD pipelines

### 🔄 Pipeline Optimization

- **Parallel processing** where possible
- **Caching strategies** for repeated operations
- **Incremental builds** using marker diffs
- **Resource optimization** for large projects
- **Failure recovery** mechanisms

### 🛡️ Security and Compliance

- **Secure storage** of markers and artifacts
- **Access control** for sensitive operations
- **Audit trails** for all deployments
- **Compliance reporting** with generated data
- **Encryption** for sensitive marker data

This comprehensive CI/CD integration guide ensures that LookAtni File Markers enhance your development workflow while maintaining security, reliability, and efficiency.
